<script lang="ts">
	interface Props {
		volume: number;
		muted: boolean;
		onVolumeChange: (volume: number) => void;
		onMuteToggle: () => void;
	}

	let { volume, muted, onVolumeChange, onMuteToggle }: Props = $props();

	let showSlider = $state(false);
</script>

<div
	class="volume-container"
	onmouseenter={() => showSlider = true}
	onmouseleave={() => showSlider = false}
>
	<button class="player-btn" onclick={onMuteToggle} aria-label={muted ? 'Unmute' : 'Mute'}>
		{#if muted || volume === 0}
			<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
				<path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51A8.796 8.796 0 0021 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06a8.99 8.99 0 003.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z" />
			</svg>
		{:else if volume < 0.5}
			<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
				<path d="M18.5 12A4.5 4.5 0 0014 7.97v8.05c2.48-.5 4.5-2.66 4.5-4.02zM5 9v6h4l5 5V4L9 9H5z" />
			</svg>
		{:else}
			<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
				<path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z" />
			</svg>
		{/if}
	</button>
	<div class="volume-slider {showSlider ? 'expanded' : ''}">
		<div class="volume-bar">
			<div class="volume-level" style="width: {muted ? 0 : volume * 100}%"></div>
		</div>
		<input
			type="range"
			min="0"
			max="1"
			step="0.05"
			value={volume}
			oninput={(e) => onVolumeChange(parseFloat(e.currentTarget.value))}
			class="volume-input"
			aria-label="Volume"
		/>
	</div>
</div>

<style>
	.volume-container {
		display: flex;
		align-items: center;
	}

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

	.volume-slider {
		width: 0;
		overflow: hidden;
		transition: width 0.2s;
		display: flex;
		align-items: center;
		position: relative;
	}

	.volume-slider.expanded {
		width: 80px;
		margin-left: 4px;
	}

	.volume-bar {
		position: absolute;
		width: 100%;
		height: 4px;
		background: rgba(245, 230, 200, 0.2);
		border-radius: 2px;
		pointer-events: none;
	}

	.volume-level {
		height: 100%;
		background: #F5E6C8;
		border-radius: 2px;
	}

	.volume-input {
		width: 100%;
		height: 4px;
		background: transparent;
		appearance: none;
		cursor: pointer;
		position: relative;
		z-index: 1;
	}

	.volume-input::-webkit-slider-thumb {
		appearance: none;
		width: 12px;
		height: 12px;
		background: #F5E6C8;
		border-radius: 50%;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
	}
</style>
