import { writable } from 'svelte/store';
import type { WorkspaceResponse, AgentResponse, WSMessage } from './types';
import { getWorkspaces, getAgents } from './api';

// Workspaces store
export const workspaces = writable<WorkspaceResponse[]>([]);
export const workspacesLoading = writable(true);
export const workspacesError = writable<string | null>(null);

// Agents store
export const agents = writable<AgentResponse[]>([]);
export const agentsLoading = writable(true);
export const agentsError = writable<string | null>(null);

// WebSocket connection status
export const wsConnected = writable(false);

// Initial data fetch
export async function loadWorkspaces() {
	workspacesLoading.set(true);
	workspacesError.set(null);
	try {
		const data = await getWorkspaces();
		workspaces.set(data);
	} catch (err) {
		workspacesError.set(err instanceof Error ? err.message : 'Failed to load workspaces');
	} finally {
		workspacesLoading.set(false);
	}
}

export async function loadAgents() {
	agentsLoading.set(true);
	agentsError.set(null);
	try {
		const data = await getAgents();
		agents.set(data);
	} catch (err) {
		agentsError.set(err instanceof Error ? err.message : 'Failed to load agents');
	} finally {
		agentsLoading.set(false);
	}
}

// WebSocket connection for real-time updates
let ws: WebSocket | null = null;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

export function connectWebSocket() {
	if (ws?.readyState === WebSocket.OPEN) {
		return;
	}

	const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	const wsUrl = `${protocol}//${window.location.host}/ws`;

	try {
		ws = new WebSocket(wsUrl);

		ws.onopen = () => {
			console.log('WebSocket connected');
			wsConnected.set(true);

			// Subscribe to updates
			ws?.send(JSON.stringify({
				type: 'subscribe',
				payload: ['workspaces', 'agents']
			}));
		};

		ws.onmessage = (event) => {
			try {
				const message: WSMessage = JSON.parse(event.data);

				switch (message.type) {
					case 'workspaces_updated':
						if (Array.isArray(message.payload)) {
							workspaces.set(message.payload as WorkspaceResponse[]);
							workspacesLoading.set(false);
						}
						break;
					case 'agents_updated':
						if (Array.isArray(message.payload)) {
							agents.set(message.payload as AgentResponse[]);
							agentsLoading.set(false);
						}
						break;
					case 'ping':
						// Keep-alive, no action needed
						break;
				}
			} catch (err) {
				console.error('Failed to parse WebSocket message:', err);
			}
		};

		ws.onclose = () => {
			console.log('WebSocket disconnected');
			wsConnected.set(false);
			ws = null;

			// Attempt to reconnect after 3 seconds
			if (!reconnectTimeout) {
				reconnectTimeout = setTimeout(() => {
					reconnectTimeout = null;
					connectWebSocket();
				}, 3000);
			}
		};

		ws.onerror = (error) => {
			console.error('WebSocket error:', error);
			wsConnected.set(false);
		};
	} catch (err) {
		console.error('Failed to create WebSocket:', err);
		wsConnected.set(false);
	}
}

export function disconnectWebSocket() {
	if (reconnectTimeout) {
		clearTimeout(reconnectTimeout);
		reconnectTimeout = null;
	}
	if (ws) {
		ws.close();
		ws = null;
	}
	wsConnected.set(false);
}
