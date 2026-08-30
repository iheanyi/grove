import type { WorkspaceResponse, AgentResponse, HealthResponse } from './types';

const API_BASE = '/api';

async function fetchJson<T>(path: string): Promise<T> {
	const response = await fetch(`${API_BASE}${path}`);
	if (!response.ok) {
		throw new Error(`API error: ${response.status} ${response.statusText}`);
	}
	return response.json();
}

export async function getWorkspaces(): Promise<WorkspaceResponse[]> {
	return fetchJson<WorkspaceResponse[]>('/workspaces');
}

export async function getAgents(): Promise<AgentResponse[]> {
	return fetchJson<AgentResponse[]>('/agents');
}

export async function getHealth(): Promise<HealthResponse> {
	return fetchJson<HealthResponse>('/health');
}
