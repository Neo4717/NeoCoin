from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import os
import random
from dotenv import load_dotenv

# Load environment variables from .env file
load_dotenv()

app = FastAPI()

# Configuration for LLM (simulated for now)
# In a real scenario, you would initialize your LLM client here
LLM_API_KEY = os.getenv("LLM_API_KEY")
LLM_PROVIDER = os.getenv("LLM_PROVIDER", "gemini") # Default to gemini

class Transaction(BaseModel):
    sender: str
    recipient: str
    amount: int
    data: str # Arbitrary data for AI to analyze

class AIRequest(BaseModel):
    transaction: Transaction

class AIResponse(BaseModel):
    valid: bool

@app.post("/audit", response_model=AIResponse)
async def audit_transaction(request: AIRequest):
    """
    Receives transaction data, analyzes it for malicious patterns using an LLM (simulated),
    and returns a boolean indicating validity.
    """
    transaction_data = request.transaction.data.lower()
    
    # Simulate LLM analysis based on transaction data
    # In a real application, you would call your LLM provider (Gemini/OpenAI) here.
    # Example using OpenAI:
    # client = OpenAI(api_key=LLM_API_KEY)
    # response = client.chat.completions.create(
    #     model="gpt-3.5-turbo",
    #     messages=[
    #         {"role": "system", "content": "You are an AI auditor. Analyze the following transaction data for malicious patterns and return 'true' if valid, 'false' if malicious."},
    #         {"role": "user", "content": f"Transaction details: Sender={request.transaction.sender}, Recipient={request.transaction.recipient}, Amount={request.transaction.amount}, Data='{request.transaction.data}'"}
    #     ]
    # )
    # is_valid = "true" in response.choices[0].message.content.lower()

    # For simulation purposes:
    if "malicious" in transaction_data or "fraud" in transaction_data or "hack" in transaction_data:
        is_valid = False
    elif "suspicious" in transaction_data and random.random() < 0.7: # 70% chance of being invalid if suspicious
        is_valid = False
    else:
        is_valid = True # Default to valid if no malicious patterns found

    print(f"Auditing transaction: {request.transaction.data} -> Valid: {is_valid}")

    return AIResponse(valid=is_valid)

@app.get("/health")
async def health_check():
    return {"status": "ok"}
