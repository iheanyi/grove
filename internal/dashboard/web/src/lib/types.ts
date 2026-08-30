// API response types matching Go backend

export interface ServerResponse {
	port: number;
	status: 'running' | 'stopped' | 'starting' | 'error';
	url: string;
	health?: string;
	started_at?: string;
	uptime?: string;
}

export interface WorkspaceResponse {
	name: string;
	path: string;
	branch: string;
	main_repo?: string;
	git_dirty: boolean;
	has_claude: boolean;
	has_vscode: boolean;
	tags?: string[];
	server?: ServerResponse;
}

export interface AgentResponse {
	worktree: string;
	path: string;
	branch: string;
	type: string;
	pid: number;
	start_time?: string;
	duration?: string;
}

export interface HealthResponse {
	status: string;
	timestamp: string;
}

// WebSocket message types
export interface WSMessage {
	type: 'workspaces_updated' | 'agents_updated' | 'ping';
	payload?: WorkspaceResponse[] | AgentResponse[];
}
