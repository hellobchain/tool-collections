import asyncio
import os
import shutil
import subprocess
import tempfile
import uuid
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, Optional

from config import (
    RESULTS_DIR, MAX_CONCURRENT_TASKS, 
    TASK_TIMEOUT, PANDOC_TIMEOUT, RESULT_RETENTION_HOURS
)
from models import TaskStatus, TaskInfo


class TaskManager:
    def __init__(self):
        self.tasks: Dict[str, TaskInfo] = {}
        self.semaphore = asyncio.Semaphore(MAX_CONCURRENT_TASKS)
        self.lock = asyncio.Lock()
        self._cleanup_task: Optional[asyncio.Task] = None
    
    async def start(self):
        """启动后台清理任务"""
        self._cleanup_task = asyncio.create_task(self._periodic_cleanup())
    
    async def stop(self):
        """停止后台任务"""
        if self._cleanup_task:
            self._cleanup_task.cancel()
            try:
                await self._cleanup_task
            except asyncio.CancelledError:
                pass
    
    async def create_task(self, markdown: str, filename: str = "document.docx",
                         reference_doc: Optional[str] = None,
                         extra_args: Optional[list] = None) -> str:
        """创建新任务"""
        task_id = str(uuid.uuid4())
        task_info = TaskInfo(
            task_id=task_id,
            status=TaskStatus.PENDING,
            created_at=datetime.utcnow(),
            message="Task queued"
        )
        
        async with self.lock:
            self.tasks[task_id] = task_info
        
        # 启动异步处理
        asyncio.create_task(self._process_task(
            task_id, markdown, filename, reference_doc, extra_args
        ))
        
        return task_id
    
    async def _process_task(self, task_id: str, markdown: str, filename: str,
                           reference_doc: Optional[str], extra_args: Optional[list]):
        """处理单个任务"""
        async with self.semaphore:
            async with self.lock:
                task = self.tasks.get(task_id)
                if not task or task.status == TaskStatus.CANCELLED:
                    return
                task.status = TaskStatus.PROCESSING
                task.started_at = datetime.utcnow()
                task.progress = 10
            
            try:
                # 执行转换（直接保存到最终路径）
                final_path = str(RESULTS_DIR / f"{task_id}_{filename}")
                await self._convert(
                    task_id, markdown, reference_doc, extra_args, output_path=final_path
                )
                
                async with self.lock:
                    task = self.tasks[task_id]
                    task.status = TaskStatus.COMPLETED
                    task.completed_at = datetime.utcnow()
                    task.progress = 100
                    task.message = "Conversion completed"
                    task.output_url = f"/download/{task_id}/{filename}"
                
            except asyncio.TimeoutError:
                async with self.lock:
                    task = self.tasks[task_id]
                    task.status = TaskStatus.FAILED
                    task.error_message = "Task timed out"
                    task.message = "Conversion timed out"
            
            except Exception as e:
                async with self.lock:
                    task = self.tasks[task_id]
                    task.status = TaskStatus.FAILED
                    task.error_message = str(e)
                    task.message = f"Conversion failed: {str(e)}"
    
    async def _convert(self, task_id: str, markdown: str, 
                      reference_doc: Optional[str], extra_args: Optional[list],
                      output_path: Optional[str] = None) -> str:
        """执行 pandoc 转换"""
        with tempfile.TemporaryDirectory() as tmpdir:
            md_path = os.path.join(tmpdir, "input.md")
            docx_path = os.path.join(tmpdir, "output.docx")
            with open(md_path, "w", encoding="utf-8") as f:
                f.write(markdown)
            
            cmd = ["pandoc", md_path, "-o", docx_path, "-f", "markdown", "-t", "docx"]
            
            if reference_doc and os.path.exists(reference_doc):
                cmd.extend(["--reference-doc", reference_doc])
            
            if extra_args:
                cmd.extend(extra_args)
            
            # 更新进度
            async with self.lock:
                if task_id in self.tasks:
                    self.tasks[task_id].progress = 50
            
            # 执行转换（带超时）
            loop = asyncio.get_event_loop()
            result = await asyncio.wait_for(
                loop.run_in_executor(
                    None,
                    lambda: subprocess.run(cmd, capture_output=True, text=True, timeout=PANDOC_TIMEOUT)
                ),
                timeout=TASK_TIMEOUT
            )
            
            if result.returncode != 0:
                raise RuntimeError(f"Pandoc failed: {result.stderr}")
            
            # 在 temp 目录清理前将文件移动到最终位置
            if output_path:
                shutil.move(docx_path, output_path)
                return output_path
            return docx_path
    
    async def get_task(self, task_id: str) -> Optional[TaskInfo]:
        """获取任务状态"""
        async with self.lock:
            return self.tasks.get(task_id)
    
    async def cancel_task(self, task_id: str) -> bool:
        """取消任务"""
        async with self.lock:
            task = self.tasks.get(task_id)
            if task and task.status in [TaskStatus.PENDING, TaskStatus.PROCESSING]:
                task.status = TaskStatus.CANCELLED
                task.message = "Task cancelled by user"
                return True
            return False
    
    async def _periodic_cleanup(self):
        """定期清理过期结果"""
        while True:
            try:
                await asyncio.sleep(1800)  # 30分钟
                
                cutoff = datetime.utcnow() - timedelta(hours=RESULT_RETENTION_HOURS)
                to_remove = []
                
                async with self.lock:
                    for task_id, task in self.tasks.items():
                        if task.completed_at and task.completed_at < cutoff:
                            to_remove.append(task_id)
                    
                    for task_id in to_remove:
                        del self.tasks[task_id]
                
                # 清理文件
                for task_id in to_remove:
                    for f in RESULTS_DIR.glob(f"{task_id}_*"):
                        try:
                            f.unlink()
                        except:
                            pass
                            
            except asyncio.CancelledError:
                break
            except Exception:
                pass


# 全局任务管理器
task_manager = TaskManager()