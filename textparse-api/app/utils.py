from urllib.parse import quote

def disposition_filename(filename: str) -> str:
    """Build Content-Disposition header value with ASCII fallback for non-latin-1 chars."""
    encoded = quote(filename, safe='')
    return f"attachment; filename=\"{encoded}\"; filename*=UTF-8''{encoded}"
