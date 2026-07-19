from fastapi import HTTPException, Security
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from config import API_KEY, ENABLE_AUTH

security = HTTPBearer(auto_error=False)


async def verify_api_key(credentials: HTTPAuthorizationCredentials = Security(security)):
    if not ENABLE_AUTH:
        return True

    if not credentials:
        raise HTTPException(
            status_code=401,
            detail="Missing API key",
        )

    if credentials.credentials != API_KEY:
        raise HTTPException(
            status_code=403,
            detail="Invalid API key",
        )

    return True
