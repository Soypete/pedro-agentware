"""Mock Kitaru server for testing."""

import uuid
from datetime import datetime
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import Any

app = FastAPI()


class FlowInput(BaseModel):
    inputs: dict[str, Any]


class CheckpointData(BaseModel):
    name: str
    data: dict[str, Any]


class ArtifactData(BaseModel):
    key: str
    data: Any
    type: str = "generic"


executions: dict[str, dict[str, Any]] = {}


@app.get("/health")
def health():
    return {"status": "ok"}


@app.post("/flows/{flow_name}/run")
def run_flow(flow_name: str, input: FlowInput):
    exec_id = str(uuid.uuid4())
    executions[exec_id] = {
        "id": exec_id,
        "status": "running",
        "started_at": datetime.now().isoformat(),
        "updated_at": datetime.now().isoformat(),
        "output": None,
        "flow_name": flow_name,
        "inputs": input.inputs,
    }
    return {"execution_id": exec_id, "status": "running"}


@app.get("/executions/{exec_id}")
def get_execution(exec_id: str):
    if exec_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")

    exec_data = executions[exec_id]
    if exec_data["status"] == "running":
        exec_data["status"] = "completed"
        exec_data["output"] = {"result": f"Flow {exec_data['flow_name']} completed successfully"}
        exec_data["updated_at"] = datetime.now().isoformat()

    return exec_data


@app.post("/executions/{exec_id}/checkpoints")
def create_checkpoint(exec_id: str, checkpoint: CheckpointData):
    if exec_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")
    return {"status": "ok", "checkpoint": checkpoint.name}


@app.get("/executions/{exec_id}/checkpoints/{checkpoint_name}")
def get_checkpoint(exec_id: str, checkpoint_name: str):
    if exec_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")
    return {"name": checkpoint_name, "data": {"saved": True}}


@app.post("/executions/{exec_id}/artifacts")
def save_artifact(exec_id: str, artifact: ArtifactData):
    if exec_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")
    return {"status": "ok", "key": artifact.key}


@app.get("/executions/{exec_id}/artifacts/{artifact_key}")
def get_artifact(exec_id: str, artifact_key: str):
    if exec_id not in executions:
        raise HTTPException(status_code=404, detail="Execution not found")
    return {"key": artifact_key, "data": {"stored": True}}