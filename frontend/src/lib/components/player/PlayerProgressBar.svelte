<script lang="ts">
	import { formatTime } from '$lib/utils';

	interface Props {
		currentTime: number;
		duration: number;
		buffered: number;
		endTime?: string;
		onSeek: (time: number) => void;
	}

	let { currentTime, duration, buffered, endTime, onSeek }: Props = $props();

	let progressBar: HTMLDivElement;
	let hoverTime = $state<number | null>(null);
	let hoverX = $state(0);

	function handleClick(e: MouseEvent) {
		const rect = progressBar.getBoundingClientRect();
		const percent = (e.clientX - rect.left) / rect.width;
		onSeek(percent * duration);
	}

	function handleHover(e: MouseEvent) {
		if (!progressBar) return;
		const rect = progressBar.getBoundingClientRect();
		const percent = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
		hoverTime = percent * duration;
		hoverX = e.clientX - rect.left;
	}

	function handleLeave() {
		hoverTime = null;
	}
</script>

<div class="progress-wrapper">
	{#if endTime}
		<div class="end-time-row">
			<span class="player-end-time">Ends {endTime}</span>
		</div>
	{/if}

	<div class="progress-row">
		<span class="player-time">{formatTime(currentTime)}</span>

		<div
			bind:this={progressBar}
			class="progress-container"
			onclick={handleClick}
			onmousemove={handleHover}
			onmouseleave={handleLeave}
			role="slider"
			aria-label="Seek"
			aria-valuemin={0}
			aria-valuemax={duration}
			aria-valuenow={currentTime}
			tabindex="0"
		>
			<div class="progress-buffered" style="width: {duration ? (buffered / duration) * 100 : 0}%"></div>
			<div class="progress-played" style="width: {duration ? (currentTime / duration) * 100 : 0}%"></div>
			<div class="progress-handle" style="left: {duration ? (currentTime / duration) * 100 : 0}%"></div>

			{#if hoverTime !== null}
				<div class="timestamp-preview" style="left: {hoverX}px">
					{formatTime(hoverTime)}
				</div>
			{/if}
		</div>

		<span class="player-time">{formatTime(duration)}</span>
	</div>
</div>

<style>
	.progress-wrapper {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.end-time-row {
		display: flex;
		justify-content: flex-end;
	}

	.player-end-time {
		color: rgba(245, 230, 200, 0.5);
		font-size: 12px;
		font-weight: 500;
	}

	.progress-row {
		display: flex;
		align-items: center;
		gap: 12px;
	}

	.player-time {
		color: rgba(245, 230, 200, 0.7);
		font-size: 13px;
		font-weight: 500;
		font-variant-numeric: tabular-nums;
		min-width: 50px;
	}

	.player-time:last-child {
		text-align: right;
	}

	.progress-container {
		flex: 1;
		position: relative;
		height: 4px;
		background: rgba(245, 230, 200, 0.2);
		border-radius: 2px;
		cursor: pointer;
		transition: height 0.2s;
	}

	.progress-container:hover {
		height: 8px;
	}

	.progress-buffered {
		position: absolute;
		height: 100%;
		background: rgba(245, 230, 200, 0.4);
		border-radius: 2px;
		pointer-events: none;
	}

	.progress-played {
		position: absolute;
		height: 100%;
		background: #E8A849;
		border-radius: 2px;
		pointer-events: none;
	}

	.progress-handle {
		position: absolute;
		width: 16px;
		height: 16px;
		background: #E8A849;
		border: 2px solid #F5E6C8;
		border-radius: 50%;
		top: 50%;
		transform: translate(-50%, -50%);
		opacity: 0;
		transition: opacity 0.2s;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
		pointer-events: none;
	}

	.progress-container:hover .progress-handle {
		opacity: 1;
	}

	.timestamp-preview {
		position: absolute;
		bottom: 100%;
		margin-bottom: 8px;
		transform: translateX(-50%);
		padding: 6px 12px;
		background: rgba(10, 10, 10, 0.9);
		backdrop-filter: blur(8px);
		border: 1px solid rgba(245, 230, 200, 0.2);
		border-radius: 6px;
		color: #F5E6C8;
		font-size: 13px;
		font-weight: 500;
		white-space: nowrap;
		pointer-events: none;
	}
</style>
