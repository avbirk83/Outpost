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

<aside class="fixed left-0 top-0 bottom-0 w-16 glass-sidebar flex flex-col z-40">
	<!-- Logo -->
	<div class="h-16 flex items-center justify-center">
		<a href="/" class="group relative flex items-center justify-center transition-all hover:scale-105" aria-label="Outpost Home">
			<img src="/logo-cream-512.png" alt="Outpost" class="w-9 h-9 object-contain" />
		</a>
	</div>

	<!-- Main Navigation -->
	<nav class="flex-1 py-3 flex flex-col gap-1 px-2">
		<!-- Home -->
		<a
			href="/"
			class="group relative flex items-center justify-center w-11 h-11 mx-auto rounded-xl transition-all duration-200
				{isActive('/') && !isActive('/library')
					? 'liquid-glass text-[#F5E6C8]'
					: 'text-[#666666] hover:text-white hover:bg-white/6'}"
			title="Home"
		>
			<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
			</svg>
			<!-- Tooltip -->
			<span class="absolute left-full ml-3 px-3 py-1.5 text-xs font-medium bg-[#111111] backdrop-blur-md border border-white/10 rounded-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50 text-white shadow-lg">
				Home
			</span>
		</a>

		<!-- Library -->
		<a
			href="/library"
			class="group relative flex items-center justify-center w-11 h-11 mx-auto rounded-xl transition-all duration-200
				{isActive('/library') || isActive('/movies') || isActive('/tv') || isActive('/music') || isActive('/books')
					? 'liquid-glass text-[#F5E6C8]'
					: 'text-[#666666] hover:text-white hover:bg-white/6'}"
			title="Library"
		>
			<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
			</svg>
			<span class="absolute left-full ml-3 px-3 py-1.5 text-xs font-medium bg-[#111111] backdrop-blur-md border border-white/10 rounded-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50 text-white shadow-lg">
				Library
			</span>
		</a>

		<!-- Discover -->
		<a
			href="/discover"
			class="group relative flex items-center justify-center w-11 h-11 mx-auto rounded-xl transition-all duration-200
				{isActive('/discover')
					? 'liquid-glass text-[#F5E6C8]'
					: 'text-[#666666] hover:text-white hover:bg-white/6'}"
			title="Discover"
		>
			<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
			</svg>
			<span class="absolute left-full ml-3 px-3 py-1.5 text-xs font-medium bg-[#111111] backdrop-blur-md border border-white/10 rounded-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50 text-white shadow-lg">
				Discover
			</span>
		</a>

		<!-- Divider -->
		<div class="my-2 mx-3 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent"></div>

		<!-- Requests -->
		<a
			href="/requests"
			class="group relative flex items-center justify-center w-11 h-11 mx-auto rounded-xl transition-all duration-200
				{isActive('/requests')
					? 'liquid-glass text-[#F5E6C8]'
					: 'text-[#666666] hover:text-white hover:bg-white/6'}"
			title="Requests"
		>
			<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
			</svg>
			<span class="absolute left-full ml-3 px-3 py-1.5 text-xs font-medium bg-[#111111] backdrop-blur-md border border-white/10 rounded-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50 text-white shadow-lg">
				Requests
			</span>
		</a>

		<!-- Admin section -->
		{#if isAdmin}
			<!-- Downloads -->
			<a
				href="/downloads"
				class="group relative flex items-center justify-center w-11 h-11 mx-auto rounded-xl transition-all duration-200
					{isActive('/downloads')
						? 'liquid-glass text-[#F5E6C8]'
						: 'text-[#666666] hover:text-white hover:bg-white/6'}"
				title="Downloads"
			>
				<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
				</svg>
				<span class="absolute left-full ml-3 px-3 py-1.5 text-xs font-medium bg-[#111111] backdrop-blur-md border border-white/10 rounded-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50 text-white shadow-lg">
					Downloads
				</span>
			</a>

			<!-- Wanted -->
			<a
				href="/wanted"
				class="group relative flex items-center justify-center w-11 h-11 mx-auto rounded-xl transition-all duration-200
					{isActive('/wanted')
						? 'liquid-glass text-[#F5E6C8]'
						: 'text-[#666666] hover:text-white hover:bg-white/6'}"
				title="Wanted"
			>
				<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
				</svg>
				<span class="absolute left-full ml-3 px-3 py-1.5 text-xs font-medium bg-[#111111] backdrop-blur-md border border-white/10 rounded-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50 text-white shadow-lg">
					Wanted
				</span>
			</a>

			<!-- Activity -->
			<a
				href="/activity"
				class="group relative flex items-center justify-center w-11 h-11 mx-auto rounded-xl transition-all duration-200
					{isActive('/activity')
						? 'liquid-glass text-[#F5E6C8]'
						: 'text-[#666666] hover:text-white hover:bg-white/6'}"
				title="Activity"
			>
				<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
				</svg>
				<span class="absolute left-full ml-3 px-3 py-1.5 text-xs font-medium bg-[#111111] backdrop-blur-md border border-white/10 rounded-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50 text-white shadow-lg">
					Activity
				</span>
			</a>
		{/if}
	</nav>

	<!-- Bottom section - Settings (admin only) -->
	{#if isAdmin}
		<div class="py-3 px-2">
			<div class="mx-3 mb-3 h-px bg-gradient-to-r from-transparent via-white/10 to-transparent"></div>
			<a
				href="/settings"
				class="group relative flex items-center justify-center w-11 h-11 mx-auto rounded-xl transition-all duration-200
					{isActive('/settings')
						? 'liquid-glass text-[#F5E6C8]'
						: 'text-[#666666] hover:text-white hover:bg-white/6'}"
				title="Settings"
			>
				<svg class="w-[18px] h-[18px]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
				<span class="absolute left-full ml-3 px-3 py-1.5 text-xs font-medium bg-[#111111] backdrop-blur-md border border-white/10 rounded-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all whitespace-nowrap z-50 text-white shadow-lg">
					Settings
				</span>
			</a>
		</div>
	{/if}
</aside>
