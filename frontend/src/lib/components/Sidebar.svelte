<script lang="ts">
	import { page } from '$app/stores';

	interface Props {
		isAdmin?: boolean;
	}

	let { isAdmin = false }: Props = $props();

	function isActive(href: string): boolean {
		if (href === '/') {
			return $page.url.pathname === '/';
		}
		return $page.url.pathname.startsWith(href);
	}
</script>

<aside class="fixed left-0 top-0 bottom-0 w-16 liquid-panel !rounded-none !border-l-0 !border-t-0 !border-b-0 flex flex-col z-40">
	<!-- Logo with glow -->
	<div class="h-16 flex items-center justify-center">
		<a href="/" class="group relative" aria-label="Outpost Home">
			<!-- Glow behind logo -->
			<div class="absolute inset-0 bg-white/20 blur-xl rounded-full opacity-0 group-hover:opacity-100 transition-opacity"></div>
			<svg class="w-8 h-8 text-white relative z-10" viewBox="0 0 24 24" fill="currentColor">
				<path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
			</svg>
		</a>
	</div>

	<!-- Main Navigation -->
	<nav class="flex-1 py-4 flex flex-col gap-1">
		<!-- Home -->
		<a
			href="/"
			class="group relative flex items-center justify-center h-12 mx-2 rounded-xl transition-all duration-200
				{isActive('/') && !isActive('/library')
					? 'liquid-glass text-white'
					: 'text-white/50 hover:text-white hover:bg-white/5'}"
			title="Home"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
			</svg>
			<!-- Active indicator -->
			{#if isActive('/') && !isActive('/library')}
				<span class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-white rounded-r"></span>
			{/if}
			<!-- Tooltip -->
			<span class="absolute left-full ml-2 liquid-badge opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50">
				Home
			</span>
		</a>

		<!-- Library -->
		<a
			href="/library"
			class="group relative flex items-center justify-center h-12 mx-2 rounded-xl transition-all duration-200
				{isActive('/library') || isActive('/movies') || isActive('/tv') || isActive('/music') || isActive('/books')
					? 'liquid-glass text-white'
					: 'text-white/50 hover:text-white hover:bg-white/5'}"
			title="Library"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
			</svg>
			{#if isActive('/library') || isActive('/movies') || isActive('/tv') || isActive('/music') || isActive('/books')}
				<span class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-white rounded-r"></span>
			{/if}
			<span class="absolute left-full ml-2 liquid-badge opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50">
				Library
			</span>
		</a>

		<!-- Discover -->
		<a
			href="/discover"
			class="group relative flex items-center justify-center h-12 mx-2 rounded-xl transition-all duration-200
				{isActive('/discover')
					? 'liquid-glass text-white'
					: 'text-white/50 hover:text-white hover:bg-white/5'}"
			title="Discover"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			{#if isActive('/discover')}
				<span class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-white rounded-r"></span>
			{/if}
			<span class="absolute left-full ml-2 liquid-badge opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50">
				Discover
			</span>
		</a>

		<!-- Requests (visible to all) -->
		<a
			href="/requests"
			class="group relative flex items-center justify-center h-12 mx-2 rounded-xl transition-all duration-200
				{isActive('/requests')
					? 'liquid-glass text-white'
					: 'text-white/50 hover:text-white hover:bg-white/5'}"
			title="Requests"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
			</svg>
			{#if isActive('/requests')}
				<span class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-white rounded-r"></span>
			{/if}
			<span class="absolute left-full ml-2 liquid-badge opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50">
				Requests
			</span>
		</a>

		<!-- Downloads (admin only) -->
		{#if isAdmin}
			<a
				href="/downloads"
				class="group relative flex items-center justify-center h-12 mx-2 rounded-xl transition-all duration-200
					{isActive('/downloads')
						? 'liquid-glass text-white'
						: 'text-white/50 hover:text-white hover:bg-white/5'}"
				title="Downloads"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
				</svg>
				{#if isActive('/downloads')}
					<span class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-white rounded-r"></span>
				{/if}
				<span class="absolute left-full ml-2 liquid-badge opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50">
					Downloads
				</span>
			</a>

			<a
				href="/wanted"
				class="group relative flex items-center justify-center h-12 mx-2 rounded-xl transition-all duration-200
					{isActive('/wanted')
						? 'liquid-glass text-white'
						: 'text-white/50 hover:text-white hover:bg-white/5'}"
				title="Wanted"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
				</svg>
				{#if isActive('/wanted')}
					<span class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-white rounded-r"></span>
				{/if}
				<span class="absolute left-full ml-2 liquid-badge opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50">
					Wanted
				</span>
			</a>
		{/if}
	</nav>

	<!-- Bottom section - Settings (admin only) -->
	{#if isAdmin}
		<div class="py-4">
			<a
				href="/settings"
				class="group relative flex items-center justify-center h-12 mx-2 rounded-xl transition-all duration-200
					{isActive('/settings')
						? 'liquid-glass text-white'
						: 'text-white/50 hover:text-white hover:bg-white/5'}"
				title="Settings"
			>
				<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
				{#if isActive('/settings')}
					<span class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-6 bg-white rounded-r"></span>
				{/if}
				<span class="absolute left-full ml-2 liquid-badge opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50">
					Settings
				</span>
			</a>
		</div>
	{/if}
</aside>
