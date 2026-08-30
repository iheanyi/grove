<script lang="ts">
	import { onMount } from 'svelte';
	import {
		agents,
		agentsLoading,
		agentsError,
		loadAgents,
		connectWebSocket,
		disconnectWebSocket
	} from '$lib/stores';

	onMount(() => {
		loadAgents();
		connectWebSocket();

		return () => {
			disconnectWebSocket();
		};
	});

	function getAgentIcon(type: string): string {
		switch (type.toLowerCase()) {
			case 'claude':
				return '🤖';
			case 'cursor':
				return '⌨️';
			case 'copilot':
				return '🧑‍✈️';
			default:
				return '🔧';
		}
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
	<title>Agents - Grove Dashboard</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h2 class="text-2xl font-bold">Active Agents</h2>
		<button class="btn btn-secondary" onclick={() => loadAgents()}>
			Refresh
		</button>
	</div>

	{#if $agentsLoading}
		<div class="card text-center py-12">
			<div class="text-slate-400">Loading agents...</div>
		</div>
	{:else if $agentsError}
		<div class="card bg-red-900/20 border-red-800">
			<div class="text-red-400">Error: {$agentsError}</div>
		</div>
	{:else if $agents.length === 0}
		<div class="card text-center py-12">
			<div class="text-4xl mb-4">🤖</div>
			<div class="text-slate-400 mb-2">No active agents found</div>
			<div class="text-sm text-slate-500">
				Agents are detected by finding running Claude Code processes
			</div>
		</div>
	{:else}
		<div class="grid gap-4">
			{#each $agents as agent (agent.pid)}
				<div class="card hover:border-slate-600 transition-colors">
					<div class="flex items-start justify-between gap-4">
						<div class="flex items-start gap-4">
							<div class="text-3xl">{getAgentIcon(agent.type)}</div>
							<div>
								<div class="flex items-center gap-3 mb-1">
									<h3 class="font-semibold text-lg">{agent.worktree}</h3>
									<span class="badge badge-green">{agent.type}</span>
								</div>
								<div class="text-sm text-slate-400 mb-2" title={agent.path}>
									{shortenPath(agent.path)}
								</div>
								<div class="flex items-center gap-4 text-sm">
									<span class="text-slate-400">
										Branch: <span class="text-slate-200">{agent.branch}</span>
									</span>
									<span class="text-slate-400">
										PID: <span class="text-slate-200">{agent.pid}</span>
									</span>
								</div>
							</div>
						</div>
						<div class="text-right">
							<div class="text-green-400 font-medium mb-1">Active</div>
							{#if agent.duration}
								<div class="text-sm text-slate-400">
									Running for {agent.duration}
								</div>
							{/if}
							{#if agent.start_time}
								<div class="text-xs text-slate-500">
									Started: {new Date(agent.start_time).toLocaleTimeString()}
								</div>
							{/if}
						</div>
					</div>
				</div>
			{/each}
		</div>

		<div class="text-sm text-slate-500 text-center">
			{$agents.length} active agent{$agents.length === 1 ? '' : 's'}
		</div>
	{/if}
</div>
