<script lang="ts">
	import { onMount } from 'svelte';
	import {
		workspaces,
		workspacesLoading,
		workspacesError,
		loadWorkspaces,
		connectWebSocket,
		disconnectWebSocket
	} from '$lib/stores';
	import type { WorkspaceResponse } from '$lib/types';

	onMount(() => {
		loadWorkspaces();
		connectWebSocket();

		return () => {
			disconnectWebSocket();
		};
	});

	function getStatusClass(workspace: WorkspaceResponse): string {
		if (!workspace.server) return 'status-stopped';
		switch (workspace.server.status) {
			case 'running':
				return 'status-running';
			case 'starting':
				return 'status-starting';
			case 'error':
				return 'status-error';
			default:
				return 'status-stopped';
		}
	}

	function getStatusText(workspace: WorkspaceResponse): string {
		if (!workspace.server) return 'No server';
		return workspace.server.status.charAt(0).toUpperCase() + workspace.server.status.slice(1);
	}

	function shortenPath(path: string): string {
		const home = '/Users/';
		if (path.startsWith(home)) {
			const rest = path.slice(home.length);
			const slashIndex = rest.indexOf('/');
			if (slashIndex !== -1) {
				return '~' + rest.slice(slashIndex);
			}
		}
		return path;
	}
</script>

<svelte:head>
	<title>Workspaces - Grove Dashboard</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h2 class="text-2xl font-bold">Workspaces</h2>
		<button class="btn btn-secondary" onclick={() => loadWorkspaces()}>
			Refresh
		</button>
	</div>

	{#if $workspacesLoading}
		<div class="card text-center py-12">
			<div class="text-slate-400">Loading workspaces...</div>
		</div>
	{:else if $workspacesError}
		<div class="card bg-red-900/20 border-red-800">
			<div class="text-red-400">Error: {$workspacesError}</div>
		</div>
	{:else if $workspaces.length === 0}
		<div class="card text-center py-12">
			<div class="text-slate-400 mb-2">No workspaces found</div>
			<div class="text-sm text-slate-500">
				Run <code class="bg-slate-700 px-1.5 py-0.5 rounded">grove discover</code> to find worktrees
			</div>
		</div>
	{:else}
		<div class="grid gap-4">
			{#each $workspaces as workspace (workspace.name)}
				<div class="card hover:border-slate-600 transition-colors">
					<div class="flex items-start justify-between gap-4">
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-3 mb-1">
								<h3 class="font-semibold text-lg truncate">{workspace.name}</h3>
								{#if workspace.git_dirty}
									<span class="badge badge-yellow">dirty</span>
								{/if}
								{#if workspace.has_claude}
									<span class="badge badge-green">claude</span>
								{/if}
								{#if workspace.has_vscode}
									<span class="badge badge-slate">vscode</span>
								{/if}
								{#if workspace.tags}
									{#each workspace.tags as tag}
										<span class="badge badge-slate">{tag}</span>
									{/each}
								{/if}
							</div>
							<div class="text-sm text-slate-400 mb-2 truncate" title={workspace.path}>
								{shortenPath(workspace.path)}
							</div>
							<div class="flex items-center gap-4 text-sm">
								<span class="text-slate-400">
									Branch: <span class="text-slate-200">{workspace.branch}</span>
								</span>
								{#if workspace.main_repo}
									<span class="text-slate-400">
										Repo: <span class="text-slate-200">{workspace.main_repo.split('/').pop()}</span>
									</span>
								{/if}
							</div>
						</div>
						<div class="text-right shrink-0">
							<div class="{getStatusClass(workspace)} font-medium mb-2">
								{getStatusText(workspace)}
							</div>
							{#if workspace.server?.url}
								<a
									href={workspace.server.url}
									target="_blank"
									rel="noopener noreferrer"
									class="text-sm text-green-400 hover:text-green-300"
								>
									{workspace.server.url}
								</a>
								{#if workspace.server.port}
									<div class="text-xs text-slate-500">
										Port {workspace.server.port}
									</div>
								{/if}
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
