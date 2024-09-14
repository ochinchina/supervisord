from fastapi import FastAPI

app = FastAPI()

@app.get("/")
async def root():
    return {"message": "Hello World"}

if __name__ == "__main__":
    import os
    for k in sorted(os.environ):
        print(k, os.environ[k])

    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=9999, reload=False)
