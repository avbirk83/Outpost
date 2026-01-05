<script lang="ts">
	type AspectRatio = 'fit' | 'fill' | '16:9' | '4:3' | '21:9';

	interface Props {
		open: boolean;
		playbackSpeed: number;
		aspectRatio: AspectRatio;
		onToggle: () => void;
		onSpeedChange: (speed: number) => void;
		onAspectChange: (ratio: AspectRatio) => void;
	}

	let { open, playbackSpeed, aspectRatio, onToggle, onSpeedChange, onAspectChange }: Props = $props();

	const speedOptions = [0.5, 0.75, 1, 1.25, 1.5, 2];
	const aspectOptions: { value: AspectRatio; label: string }[] = [
		{ value: 'fit', label: 'Fit' },
		{ value: 'fill', label: 'Fill' },
		{ value: '16:9', label: '16:9' },
		{ value: '4:3', label: '4:3' },
		{ value: '21:9', label: '21:9 (Ultrawide)' }
	];
</script>

<div class="relative settings-menu">
	<button
		class="player-btn"
		onclick={onToggle}
		aria-label="Settings"
	>
		<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
			<path d="M19.14 12.94c.04-.31.06-.63.06-.94 0-.31-.02-.63-.06-.94l2.03-1.58c.18-.14.23-.41.12-.61l-1.92-3.32c-.12-.22-.37-.29-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54c-.04-.24-.24-.41-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.04.31-.06.63-.06.94s.02.63.06.94l-2.03 1.58c-.18.14-.23.41-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6c-1.98 0-3.6-1.62-3.6-3.6s1.62-3.6 3.6-3.6 3.6 1.62 3.6 3.6-1.62 3.6-3.6 3.6z"/>
		</svg>
	</button>

	{#if open}
		<div class="settings-dropdown">
			<div class="settings-header">Playback Speed</div>
			{#each speedOptions as speed}
				<button
					class="settings-item {playbackSpeed === speed ? 'active' : ''}"
					onclick={() => onSpeedChange(speed)}
				>
					<span>{speed === 1 ? 'Normal' : speed + 'x'}</span>
					{#if playbackSpeed === speed}
						<svg class="w-4 h-4 text-amber" fill="currentColor" viewBox="0 0 24 24">
							<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
						</svg>
					{/if}
				</button>
			{/each}

			<div class="settings-divider"></div>
			<div class="settings-header">Aspect Ratio</div>
			{#each aspectOptions as option}
				<button
					class="settings-item {aspectRatio === option.value ? 'active' : ''}"
					onclick={() => onAspectChange(option.value)}
				>
					<span>{option.label}</span>
					{#if aspectRatio === option.value}
						<svg class="w-4 h-4 text-amber" fill="currentColor" viewBox="0 0 24 24">
							<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
						</svg>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.player-btn {
		background: none;
		border: none;
		color: #F5E6C8;
		padding: 10px;
		cursor: pointer;
		border-radius: 50%;
		transition: all 0.2s;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.player-btn:hover {
		color: #E8A849;
		background: rgba(255, 255, 255, 0.06);
	}

	.settings-dropdown {
		position: absolute;
		bottom: 100%;
		right: 0;
		margin-bottom: 12px;
		min-width: 200px;
		max-height: 300px;
		overflow-y: auto;
		background: rgba(10, 10, 10, 0.95);
		backdrop-filter: blur(20px);
		border: 1px solid rgba(245, 230, 200, 0.1);
		border-radius: 12px;
		padding: 8px 0;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
	}

	.settings-header {
		padding: 8px 16px;
		color: rgba(245, 230, 200, 0.5);
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.settings-divider {
		height: 1px;
		background: rgba(245, 230, 200, 0.1);
		margin: 8px 0;
	}

	.settings-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 12px 16px;
		color: #F5E6C8;
		font-size: 14px;
		cursor: pointer;
		transition: background 0.2s;
		background: none;
		border: none;
		width: 100%;
		text-align: left;
	}

	.settings-item:hover {
		background: rgba(255, 255, 255, 0.06);
	}

	.settings-item.active {
		color: #E8A849;
	}

	.text-amber {
		color: #E8A849;
	}
</style>
