<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';

	let { children } = $props();

	const navItems = [
		{ href: '/', label: 'Workspaces', icon: '📁' },
		{ href: '/agents', label: 'Agents', icon: '🤖' }
	];
</script>

<div class="min-h-screen flex flex-col">
	<!-- Header -->
	<header class="bg-slate-800 border-b border-slate-700">
		<div class="max-w-7xl mx-auto px-4 py-3">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-6">
					<h1 class="text-xl font-bold text-green-400 flex items-center gap-2">
						<span class="text-2xl">🌲</span>
						Grove
					</h1>
					<nav class="flex gap-4">
						{#each navItems as item}
							<a
								href={item.href}
								class="px-3 py-1.5 rounded-md text-sm font-medium transition-colors
									{$page.url.pathname === item.href
										? 'bg-slate-700 text-white'
										: 'text-slate-400 hover:text-white hover:bg-slate-700/50'}"
							>
								<span class="mr-1.5">{item.icon}</span>
								{item.label}
							</a>
						{/each}
					</nav>
				</div>
				<div class="flex items-center gap-3 text-sm text-slate-400">
					<span id="connection-status" class="flex items-center gap-1.5">
						<span class="w-2 h-2 rounded-full bg-green-400"></span>
						Connected
					</span>
				</div>
			</div>
		</div>
	</header>

	<!-- Main content -->
	<main class="flex-1 max-w-7xl mx-auto px-4 py-6 w-full">
		{@render children()}
	</main>

	<!-- Footer -->
	<footer class="bg-slate-800 border-t border-slate-700">
		<div class="max-w-7xl mx-auto px-4 py-3 text-center text-sm text-slate-500">
			Grove Dashboard - Worktree Server Manager
		</div>
	</footer>
</div>
