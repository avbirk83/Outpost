<script lang="ts">
	import { formatTime } from '$lib/utils';
	import type { Chapter } from '$lib/api';

	interface Props {
		currentTime: number;
		duration: number;
		buffered: number;
		chapters?: Chapter[];
		endTime?: string;
		showRemaining?: boolean;
		onSeek: (time: number) => void;
		onToggleTimeDisplay?: () => void;
	}

	let { currentTime, duration, buffered, chapters = [], endTime, showRemaining = false, onSeek, onToggleTimeDisplay }: Props = $props();

	function handleTimeClick() {
		if (onToggleTimeDisplay) {
			onToggleTimeDisplay();
		}
	}

	let progressBar: HTMLDivElement;
	let hoverTime = $state<number | null>(null);
	let hoverX = $state(0);
	let hoverChapter = $state<string | null>(null);

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

		// Find chapter at hover time
		if (chapters && chapters.length > 0 && hoverTime !== null) {
			const chapter = chapters.find(c => hoverTime! >= c.startTime && hoverTime! < c.endTime);
			hoverChapter = chapter?.title || null;
		} else {
			hoverChapter = null;
		}
	}

	function handleLeave() {
		hoverTime = null;
		hoverChapter = null;
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

			{#if chapters && chapters.length > 0}
				{#each chapters as chapter}
					{#if chapter.startTime > 0}
						<div class="chapter-marker" style="left: {(chapter.startTime / duration) * 100}%"></div>
					{/if}
				{/each}
			{/if}

			{#if hoverTime !== null}
				<div class="timestamp-preview" style="left: {hoverX}px">
					<span class="preview-time">{formatTime(hoverTime)}</span>
					{#if hoverChapter}
						<span class="preview-chapter">{hoverChapter}</span>
					{/if}
				</div>
			{/if}
		</div>

		<button class="player-time player-time-clickable" onclick={handleTimeClick} title="Click to toggle remaining time">
			{#if showRemaining}
				-{formatTime(duration - currentTime)}
			{:else}
				{formatTime(duration)}
			{/if}
		</button>
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

	.player-time-clickable {
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px 8px;
		margin: -4px -8px;
		border-radius: 4px;
		transition: background 0.2s;
	}

	.player-time-clickable:hover {
		background: rgba(255, 255, 255, 0.1);
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
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
	}

	.preview-time {
		font-variant-numeric: tabular-nums;
	}

	.preview-chapter {
		font-size: 11px;
		color: #E8A849;
		max-width: 200px;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.chapter-marker {
		position: absolute;
		top: 0;
		bottom: 0;
		width: 3px;
		background: rgba(245, 230, 200, 0.6);
		transform: translateX(-50%);
		pointer-events: none;
		border-radius: 1px;
	}

	.progress-container:hover .chapter-marker {
		background: rgba(245, 230, 200, 0.8);
	}
</style>
