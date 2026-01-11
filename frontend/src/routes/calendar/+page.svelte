<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getCalendarItems, getTmdbImageUrl, getImageUrl, type CalendarItem, type CalendarFilter } from '$lib/api';

	// State
	let currentDate = $state(new Date());
	let items = $state<CalendarItem[]>([]);
	let loading = $state(true);
	let filter = $state<CalendarFilter>('all');
	let view = $state<'month' | 'week' | 'agenda'>('month');
	let selectedDay = $state<string | null>(null);

	// Computed values
	let currentMonth = $derived(currentDate.getMonth());
	let currentYear = $derived(currentDate.getFullYear());
	let monthName = $derived(currentDate.toLocaleString('default', { month: 'long' }));

	// Get first and last day of month for API
	let startDate = $derived(new Date(currentYear, currentMonth, 1));
	let endDate = $derived(new Date(currentYear, currentMonth + 1, 0));

	// Get days in month grid (including padding from previous/next months)
	let calendarDays = $derived(getCalendarDays(currentYear, currentMonth));

	// Group items by date
	let itemsByDate = $derived(groupItemsByDate(items));

	// Items for selected day
	let selectedDayItems = $derived(selectedDay ? itemsByDate.get(selectedDay) || [] : []);

	// Week view days
	let weekDays = $derived(getWeekDays());

	function getWeekDays(): Date[] {
		const today = new Date();
		const startOfWeek = new Date(today);
		startOfWeek.setDate(today.getDate() - today.getDay());

		return Array(7).fill(null).map((_, i) => {
			const day = new Date(startOfWeek);
			day.setDate(startOfWeek.getDate() + i);
			return day;
		});
	}

	function getCalendarDays(year: number, month: number): Date[] {
		const days: Date[] = [];
		const firstDay = new Date(year, month, 1);
		const lastDay = new Date(year, month + 1, 0);

		// Start from Sunday of the week containing the first day
		const startPadding = firstDay.getDay();
		for (let i = startPadding - 1; i >= 0; i--) {
			days.push(new Date(year, month, -i));
		}

		// Add all days of the month
		for (let d = 1; d <= lastDay.getDate(); d++) {
			days.push(new Date(year, month, d));
		}

		// Add days from next month to complete the grid (6 rows)
		const remaining = 42 - days.length;
		for (let i = 1; i <= remaining; i++) {
			days.push(new Date(year, month + 1, i));
		}

		return days;
	}

	function groupItemsByDate(items: CalendarItem[]): Map<string, CalendarItem[]> {
		const map = new Map<string, CalendarItem[]>();
		for (const item of items) {
			const existing = map.get(item.date) || [];
			existing.push(item);
			map.set(item.date, existing);
		}
		return map;
	}

	function formatDateKey(date: Date): string {
		return date.toISOString().split('T')[0];
	}

	function isToday(date: Date): boolean {
		const today = new Date();
		return date.getDate() === today.getDate() &&
			date.getMonth() === today.getMonth() &&
			date.getFullYear() === today.getFullYear();
	}

	function isCurrentMonth(date: Date): boolean {
		return date.getMonth() === currentMonth;
	}

	function previousMonth() {
		currentDate = new Date(currentYear, currentMonth - 1, 1);
	}

	function nextMonth() {
		currentDate = new Date(currentYear, currentMonth + 1, 1);
	}

	function goToToday() {
		currentDate = new Date();
	}

	function selectDay(date: Date) {
		const key = formatDateKey(date);
		if (selectedDay === key) {
			selectedDay = null;
		} else {
			selectedDay = key;
		}
	}

	function getItemLink(item: CalendarItem): string {
		if (item.type === 'episode') {
			if (item.mediaId) {
				return `/tv/${item.mediaId}`;
			}
			return `/explore/show/${item.tmdbId}`;
		} else {
			if (item.mediaId) {
				return `/movies/${item.mediaId}`;
			}
			return `/explore/movie/${item.tmdbId}`;
		}
	}

	// Get poster image URL - use local cache for library items, TMDB for others
	function getPosterUrl(item: CalendarItem, size: string = 'w92'): string {
		if (!item.posterPath) return '';
		// Library items have locally cached images
		if (item.inLibrary) {
			return getImageUrl(item.posterPath) || '';
		}
		// Wanted/external items use direct TMDB URLs
		return getTmdbImageUrl(item.posterPath, size);
	}

	async function loadItems() {
		loading = true;
		try {
			const start = formatDateKey(startDate);
			const end = formatDateKey(endDate);
			items = await getCalendarItems(start, end, filter);
		} catch (e) {
			console.error('Failed to load calendar:', e);
		}
		loading = false;
	}

	// Load on mount and when dependencies change
	onMount(() => {
		// Check for month param in URL
		const monthParam = $page.url.searchParams.get('month');
		if (monthParam) {
			const [year, month] = monthParam.split('-').map(Number);
			if (year && month) {
				currentDate = new Date(year, month - 1, 1);
			}
		}
	});

	$effect(() => {
		// Reload when date or filter changes
		const _ = [startDate, endDate, filter];
		loadItems();
	});
</script>

<svelte:head>
	<title>Calendar - Outpost</title>
</svelte:head>

<div class="calendar-page">
	<!-- Header -->
	<div class="calendar-header">
		<div class="header-left">
			<h1>{monthName} {currentYear}</h1>
			<div class="nav-buttons">
				<button class="nav-btn" onclick={previousMonth} aria-label="Previous month">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<button class="today-btn" onclick={goToToday}>Today</button>
				<button class="nav-btn" onclick={nextMonth} aria-label="Next month">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			</div>
		</div>

		<div class="header-right">
			<!-- Filter pills -->
			<div class="filter-pills">
				{#each ['all', 'movies', 'tv', 'library', 'wanted'] as f}
					<button
						class="filter-pill {filter === f ? 'active' : ''}"
						onclick={() => filter = f as CalendarFilter}
					>
						{f === 'all' ? 'All' : f === 'movies' ? 'Movies' : f === 'tv' ? 'TV' : f === 'library' ? 'In Library' : 'Wanted'}
					</button>
				{/each}
			</div>

			<!-- View toggle -->
			<div class="view-toggle">
				<button class="view-btn {view === 'month' ? 'active' : ''}" onclick={() => view = 'month'}>Month</button>
				<button class="view-btn {view === 'week' ? 'active' : ''}" onclick={() => view = 'week'}>Week</button>
				<button class="view-btn {view === 'agenda' ? 'active' : ''}" onclick={() => view = 'agenda'}>Agenda</button>
			</div>
		</div>
	</div>

	{#if loading}
		<div class="loading">
			<div class="loading-spinner"></div>
		</div>
	{:else if view === 'month'}
		<!-- Month View -->
		<div class="month-view">
			<!-- Day headers -->
			<div class="day-headers">
				{#each ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'] as day}
					<div class="day-header">{day}</div>
				{/each}
			</div>

			<!-- Calendar grid -->
			<div class="calendar-grid">
				{#each calendarDays as day}
					{@const dateKey = formatDateKey(day)}
					{@const dayItems = itemsByDate.get(dateKey) || []}
					<button
						class="calendar-day {isToday(day) ? 'today' : ''} {!isCurrentMonth(day) ? 'other-month' : ''} {selectedDay === dateKey ? 'selected' : ''} {dayItems.length > 0 ? 'has-items' : ''}"
						onclick={() => selectDay(day)}
					>
						<span class="day-number">{day.getDate()}</span>
						{#if dayItems.length > 0}
							<div class="day-items">
								{#each dayItems.slice(0, 3) as item}
									<div class="day-item {item.type}" title="{item.title}: {item.subtitle}">
										{#if item.posterPath}
											<img src={getPosterUrl(item, 'w92')} alt="" class="item-poster" />
										{:else}
											<div class="item-poster-placeholder {item.type}"></div>
										{/if}
									</div>
								{/each}
								{#if dayItems.length > 3}
									<span class="more-count">+{dayItems.length - 3}</span>
								{/if}
							</div>
						{/if}
					</button>
				{/each}
			</div>
		</div>
	{:else if view === 'week'}
		<!-- Week View -->
		<div class="week-view">
			<div class="week-grid">
				{#each weekDays as day}
					{@const dateKey = formatDateKey(day)}
					{@const dayItems = itemsByDate.get(dateKey) || []}
					<div class="week-day {isToday(day) ? 'today' : ''}">
						<div class="week-day-header">
							<span class="week-day-name">{day.toLocaleDateString('default', { weekday: 'short' })}</span>
							<span class="week-day-num">{day.getDate()}</span>
						</div>
						<div class="week-day-items">
							{#each dayItems as item}
								<a href={getItemLink(item)} class="week-item {item.type}">
									{#if item.posterPath}
										<img src={getPosterUrl(item, 'w92')} alt="" class="week-item-poster" />
									{/if}
									<div class="week-item-info">
										<span class="week-item-title">{item.title}</span>
										<span class="week-item-subtitle">{item.subtitle}</span>
									</div>
								</a>
							{/each}
							{#if dayItems.length === 0}
								<div class="no-items">No releases</div>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{:else}
		<!-- Agenda View -->
		<div class="agenda-view">
			{#if items.length === 0}
				<div class="empty-state">
					<svg class="w-16 h-16" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
					</svg>
					<h3>Nothing scheduled</h3>
					<p>No releases for this period. Try adding shows to your library or wanted list.</p>
				</div>
			{:else}
				{@const sortedDates = [...itemsByDate.keys()].sort()}
				{#each sortedDates as dateKey}
					{@const dayItems = itemsByDate.get(dateKey) || []}
					{@const date = new Date(dateKey + 'T00:00:00')}
					<div class="agenda-day">
						<div class="agenda-date">
							<span class="agenda-weekday">{date.toLocaleDateString('default', { weekday: 'long' })}</span>
							<span class="agenda-full-date">{date.toLocaleDateString('default', { month: 'long', day: 'numeric', year: 'numeric' })}</span>
						</div>
						<div class="agenda-items">
							{#each dayItems as item}
								<a href={getItemLink(item)} class="agenda-item">
									{#if item.posterPath}
										<img src={getPosterUrl(item, 'w154')} alt="" class="agenda-poster" />
									{:else}
										<div class="agenda-poster-placeholder {item.type}"></div>
									{/if}
									<div class="agenda-item-info">
										<span class="agenda-item-title">{item.title}</span>
										<span class="agenda-item-subtitle">{item.subtitle}</span>
										<div class="agenda-item-badges">
											<span class="type-badge {item.type}">{item.type === 'episode' ? 'TV' : 'Movie'}</span>
											{#if item.inLibrary}
												<span class="status-badge library">In Library</span>
											{:else if item.isWanted}
												<span class="status-badge wanted">Wanted</span>
											{/if}
										</div>
									</div>
								</a>
							{/each}
						</div>
					</div>
				{/each}
			{/if}
		</div>
	{/if}

	<!-- Day detail modal -->
	{#if selectedDay && selectedDayItems.length > 0}
		<div class="day-modal-overlay" onclick={() => selectedDay = null} role="dialog">
			<div class="day-modal" onclick={(e) => e.stopPropagation()}>
				<div class="day-modal-header">
					<h2>{new Date(selectedDay + 'T00:00:00').toLocaleDateString('default', { weekday: 'long', month: 'long', day: 'numeric' })}</h2>
					<button class="close-btn" onclick={() => selectedDay = null} aria-label="Close">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
				<div class="day-modal-items">
					{#each selectedDayItems as item}
						<a href={getItemLink(item)} class="modal-item" onclick={() => selectedDay = null}>
							{#if item.posterPath}
								<img src={getPosterUrl(item, 'w154')} alt="" class="modal-poster" />
							{:else}
								<div class="modal-poster-placeholder {item.type}"></div>
							{/if}
							<div class="modal-item-info">
								<span class="modal-item-title">{item.title}</span>
								<span class="modal-item-subtitle">{item.subtitle}</span>
								<div class="modal-item-badges">
									<span class="type-badge {item.type}">{item.type === 'episode' ? 'TV' : 'Movie'}</span>
									{#if item.inLibrary}
										<span class="status-badge library">In Library</span>
									{:else if item.isWanted}
										<span class="status-badge wanted">Wanted</span>
									{/if}
									{#if item.airTime}
										<span class="time-badge">{item.airTime}</span>
									{/if}
								</div>
							</div>
						</a>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.calendar-page {
		max-width: 1400px;
		margin: 0 auto;
		padding: 24px;
	}

	.calendar-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 24px;
		flex-wrap: wrap;
		gap: 16px;
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 24px;
	}

	.header-left h1 {
		font-size: 28px;
		font-weight: 600;
		color: #F5E6C8;
		margin: 0;
	}

	.nav-buttons {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.nav-btn {
		width: 36px;
		height: 36px;
		border-radius: 8px;
		border: 1px solid rgba(245, 230, 200, 0.2);
		background: rgba(255, 255, 255, 0.05);
		color: #F5E6C8;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.2s;
	}

	.nav-btn:hover {
		background: rgba(255, 255, 255, 0.1);
		border-color: rgba(245, 230, 200, 0.3);
	}

	.today-btn {
		padding: 8px 16px;
		border-radius: 8px;
		border: 1px solid rgba(245, 230, 200, 0.2);
		background: rgba(255, 255, 255, 0.05);
		color: #F5E6C8;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
	}

	.today-btn:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.header-right {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.filter-pills {
		display: flex;
		gap: 8px;
	}

	.filter-pill {
		padding: 6px 12px;
		border-radius: 16px;
		border: 1px solid rgba(245, 230, 200, 0.2);
		background: transparent;
		color: rgba(245, 230, 200, 0.7);
		font-size: 13px;
		cursor: pointer;
		transition: all 0.2s;
	}

	.filter-pill:hover {
		background: rgba(255, 255, 255, 0.05);
	}

	.filter-pill.active {
		background: #E8A849;
		border-color: #E8A849;
		color: #1a1a1a;
	}

	.view-toggle {
		display: flex;
		background: rgba(255, 255, 255, 0.05);
		border-radius: 8px;
		padding: 4px;
	}

	.view-btn {
		padding: 6px 12px;
		border-radius: 6px;
		border: none;
		background: transparent;
		color: rgba(245, 230, 200, 0.7);
		font-size: 13px;
		cursor: pointer;
		transition: all 0.2s;
	}

	.view-btn.active {
		background: rgba(255, 255, 255, 0.1);
		color: #F5E6C8;
	}

	/* Loading */
	.loading {
		display: flex;
		justify-content: center;
		align-items: center;
		height: 400px;
	}

	.loading-spinner {
		width: 40px;
		height: 40px;
		border: 3px solid rgba(245, 230, 200, 0.2);
		border-top-color: #E8A849;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Month View */
	.month-view {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 12px;
		overflow: hidden;
	}

	.day-headers {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		background: rgba(255, 255, 255, 0.03);
		border-bottom: 1px solid rgba(245, 230, 200, 0.1);
	}

	.day-header {
		padding: 12px;
		text-align: center;
		font-size: 12px;
		font-weight: 600;
		color: rgba(245, 230, 200, 0.5);
		text-transform: uppercase;
	}

	.calendar-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
	}

	.calendar-day {
		min-height: 100px;
		padding: 8px;
		border: none;
		border-right: 1px solid rgba(245, 230, 200, 0.05);
		border-bottom: 1px solid rgba(245, 230, 200, 0.05);
		background: transparent;
		cursor: pointer;
		transition: background 0.2s;
		text-align: left;
		display: flex;
		flex-direction: column;
	}

	.calendar-day:nth-child(7n) {
		border-right: none;
	}

	.calendar-day:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	.calendar-day.selected {
		background: rgba(232, 168, 73, 0.1);
	}

	.calendar-day.other-month {
		opacity: 0.4;
	}

	.calendar-day.today .day-number {
		background: #E8A849;
		color: #1a1a1a;
		border-radius: 50%;
		width: 28px;
		height: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.day-number {
		font-size: 14px;
		font-weight: 500;
		color: #F5E6C8;
		margin-bottom: 4px;
	}

	.day-items {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-top: auto;
	}

	.day-item {
		width: 30px;
		height: 45px;
		border-radius: 4px;
		overflow: hidden;
	}

	.day-item.movie {
		border: 2px solid rgba(232, 168, 73, 0.5);
	}

	.day-item.episode {
		border: 2px solid rgba(59, 130, 246, 0.5);
	}

	.item-poster {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.item-poster-placeholder {
		width: 100%;
		height: 100%;
	}

	.item-poster-placeholder.movie {
		background: rgba(232, 168, 73, 0.2);
	}

	.item-poster-placeholder.episode {
		background: rgba(59, 130, 246, 0.2);
	}

	.more-count {
		font-size: 11px;
		color: rgba(245, 230, 200, 0.5);
		align-self: flex-end;
	}

	/* Week View */
	.week-view {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 12px;
		overflow: hidden;
	}

	.week-grid {
		display: grid;
		grid-template-columns: repeat(7, 1fr);
	}

	.week-day {
		min-height: 400px;
		border-right: 1px solid rgba(245, 230, 200, 0.05);
		display: flex;
		flex-direction: column;
	}

	.week-day:last-child {
		border-right: none;
	}

	.week-day.today .week-day-header {
		background: rgba(232, 168, 73, 0.1);
	}

	.week-day-header {
		padding: 12px;
		text-align: center;
		border-bottom: 1px solid rgba(245, 230, 200, 0.05);
		background: rgba(255, 255, 255, 0.03);
	}

	.week-day-name {
		font-size: 12px;
		color: rgba(245, 230, 200, 0.5);
		display: block;
	}

	.week-day-num {
		font-size: 18px;
		font-weight: 600;
		color: #F5E6C8;
	}

	.week-day-items {
		padding: 8px;
		flex: 1;
		overflow-y: auto;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.week-item {
		display: flex;
		gap: 8px;
		padding: 8px;
		background: rgba(255, 255, 255, 0.03);
		border-radius: 8px;
		text-decoration: none;
		transition: background 0.2s;
	}

	.week-item:hover {
		background: rgba(255, 255, 255, 0.06);
	}

	.week-item.movie {
		border-left: 3px solid #E8A849;
	}

	.week-item.episode {
		border-left: 3px solid #3b82f6;
	}

	.week-item-poster {
		width: 40px;
		height: 60px;
		object-fit: cover;
		border-radius: 4px;
	}

	.week-item-info {
		flex: 1;
		min-width: 0;
	}

	.week-item-title {
		font-size: 13px;
		font-weight: 500;
		color: #F5E6C8;
		display: block;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.week-item-subtitle {
		font-size: 11px;
		color: rgba(245, 230, 200, 0.5);
		display: block;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.no-items {
		text-align: center;
		color: rgba(245, 230, 200, 0.3);
		font-size: 12px;
		padding: 20px;
	}

	/* Agenda View */
	.agenda-view {
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.empty-state {
		text-align: center;
		padding: 60px 20px;
		color: rgba(245, 230, 200, 0.5);
	}

	.empty-state svg {
		margin: 0 auto 16px;
		opacity: 0.3;
	}

	.empty-state h3 {
		font-size: 18px;
		font-weight: 600;
		color: #F5E6C8;
		margin: 0 0 8px;
	}

	.empty-state p {
		margin: 0;
		font-size: 14px;
	}

	.agenda-day {
		background: rgba(255, 255, 255, 0.02);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 12px;
		overflow: hidden;
	}

	.agenda-date {
		padding: 12px 16px;
		background: rgba(255, 255, 255, 0.03);
		border-bottom: 1px solid rgba(245, 230, 200, 0.1);
		display: flex;
		gap: 12px;
		align-items: baseline;
	}

	.agenda-weekday {
		font-size: 14px;
		font-weight: 600;
		color: #E8A849;
	}

	.agenda-full-date {
		font-size: 14px;
		color: rgba(245, 230, 200, 0.5);
	}

	.agenda-items {
		display: flex;
		flex-direction: column;
	}

	.agenda-item {
		display: flex;
		gap: 16px;
		padding: 16px;
		text-decoration: none;
		transition: background 0.2s;
		border-bottom: 1px solid rgba(245, 230, 200, 0.05);
	}

	.agenda-item:last-child {
		border-bottom: none;
	}

	.agenda-item:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	.agenda-poster {
		width: 60px;
		height: 90px;
		object-fit: cover;
		border-radius: 6px;
	}

	.agenda-poster-placeholder {
		width: 60px;
		height: 90px;
		border-radius: 6px;
	}

	.agenda-poster-placeholder.movie {
		background: rgba(232, 168, 73, 0.2);
	}

	.agenda-poster-placeholder.episode {
		background: rgba(59, 130, 246, 0.2);
	}

	.agenda-item-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.agenda-item-title {
		font-size: 16px;
		font-weight: 500;
		color: #F5E6C8;
	}

	.agenda-item-subtitle {
		font-size: 14px;
		color: rgba(245, 230, 200, 0.5);
	}

	.agenda-item-badges {
		display: flex;
		gap: 8px;
		margin-top: 8px;
	}

	.type-badge {
		padding: 3px 8px;
		border-radius: 4px;
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
	}

	.type-badge.movie {
		background: rgba(232, 168, 73, 0.2);
		color: #E8A849;
	}

	.type-badge.episode {
		background: rgba(59, 130, 246, 0.2);
		color: #3b82f6;
	}

	.status-badge {
		padding: 3px 8px;
		border-radius: 4px;
		font-size: 11px;
		font-weight: 500;
	}

	.status-badge.library {
		background: rgba(34, 197, 94, 0.2);
		color: #22c55e;
	}

	.status-badge.wanted {
		background: rgba(232, 168, 73, 0.2);
		color: #E8A849;
	}

	.time-badge {
		padding: 3px 8px;
		border-radius: 4px;
		font-size: 11px;
		background: rgba(255, 255, 255, 0.1);
		color: rgba(245, 230, 200, 0.7);
	}

	/* Day Modal */
	.day-modal-overlay {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.7);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 100;
		padding: 20px;
	}

	.day-modal {
		background: #1a1a1a;
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 16px;
		width: 100%;
		max-width: 500px;
		max-height: 80vh;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}

	.day-modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 16px 20px;
		border-bottom: 1px solid rgba(245, 230, 200, 0.1);
	}

	.day-modal-header h2 {
		font-size: 18px;
		font-weight: 600;
		color: #F5E6C8;
		margin: 0;
	}

	.close-btn {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		border: none;
		background: rgba(255, 255, 255, 0.05);
		color: #F5E6C8;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: background 0.2s;
	}

	.close-btn:hover {
		background: rgba(255, 255, 255, 0.1);
	}

	.day-modal-items {
		overflow-y: auto;
		flex: 1;
	}

	.modal-item {
		display: flex;
		gap: 16px;
		padding: 16px 20px;
		text-decoration: none;
		transition: background 0.2s;
		border-bottom: 1px solid rgba(245, 230, 200, 0.05);
	}

	.modal-item:last-child {
		border-bottom: none;
	}

	.modal-item:hover {
		background: rgba(255, 255, 255, 0.03);
	}

	.modal-poster {
		width: 60px;
		height: 90px;
		object-fit: cover;
		border-radius: 6px;
	}

	.modal-poster-placeholder {
		width: 60px;
		height: 90px;
		border-radius: 6px;
	}

	.modal-poster-placeholder.movie {
		background: rgba(232, 168, 73, 0.2);
	}

	.modal-poster-placeholder.episode {
		background: rgba(59, 130, 246, 0.2);
	}

	.modal-item-info {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.modal-item-title {
		font-size: 16px;
		font-weight: 500;
		color: #F5E6C8;
	}

	.modal-item-subtitle {
		font-size: 14px;
		color: rgba(245, 230, 200, 0.5);
	}

	.modal-item-badges {
		display: flex;
		gap: 8px;
		margin-top: 8px;
	}

	/* Responsive */
	@media (max-width: 768px) {
		.calendar-page {
			padding: 16px;
		}

		.calendar-header {
			flex-direction: column;
			align-items: flex-start;
		}

		.header-right {
			flex-direction: column;
			width: 100%;
			gap: 12px;
		}

		.filter-pills {
			flex-wrap: wrap;
		}

		.calendar-day {
			min-height: 80px;
			padding: 4px;
		}

		.day-number {
			font-size: 12px;
		}

		.day-items {
			gap: 2px;
		}

		.day-item {
			width: 24px;
			height: 36px;
		}

		.week-grid {
			grid-template-columns: 1fr;
		}

		.week-day {
			min-height: auto;
			border-right: none;
			border-bottom: 1px solid rgba(245, 230, 200, 0.1);
		}
	}
</style>
