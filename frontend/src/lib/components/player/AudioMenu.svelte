<script lang="ts">
	interface AudioTrack {
		index: number;
		language: string;
		label: string;
	}

	interface Props {
		tracks: AudioTrack[];
		selectedIndex: number;
		audioSync: number;
		open: boolean;
		onToggle: () => void;
		onTrackSelect: (index: number) => void;
		onSyncChange: (delta: number) => void;
		onSyncReset: () => void;
	}

	let { tracks, selectedIndex, audioSync, open, onToggle, onTrackSelect, onSyncChange, onSyncReset }: Props = $props();

	function getTrackLabel(track: AudioTrack): string {
		const langNames: Record<string, string> = {
			'eng': 'English', 'en': 'English',
			'spa': 'Spanish', 'es': 'Spanish',
			'fra': 'French', 'fr': 'French',
			'deu': 'German', 'de': 'German',
			'ita': 'Italian', 'it': 'Italian',
			'jpn': 'Japanese', 'ja': 'Japanese',
			'kor': 'Korean', 'ko': 'Korean',
			'por': 'Portuguese', 'pt': 'Portuguese',
			'rus': 'Russian', 'ru': 'Russian',
			'zho': 'Chinese', 'zh': 'Chinese',
			'und': 'Unknown'
		};

		if (track.label && track.label !== `Track ${track.index + 1}`) {
			return track.label;
		}

		return langNames[track.language] || track.language || `Track ${track.index + 1}`;
	}
</script>

<div class="relative audio-menu">
	<button
		class="player-btn"
		onclick={onToggle}
		aria-label="Audio settings"
	>
		<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
			<path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
		</svg>
	</button>

	{#if open}
		<div class="track-dropdown">
			{#if tracks.length > 1}
				<div class="settings-header">Audio Track</div>
				{#each tracks as track (track.index)}
					<button
						class="track-item {selectedIndex === track.index ? 'active' : ''}"
						onclick={() => onTrackSelect(track.index)}
					>
						<span class="track-item-check">
							{#if selectedIndex === track.index}
								<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
									<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
								</svg>
							{/if}
						</span>
						<span class="track-item-label">{getTrackLabel(track)}</span>
					</button>
				{/each}
				<div class="settings-divider"></div>
			{/if}

			<div class="settings-header">Audio Sync</div>
			<div class="audio-sync-control">
				<button class="sync-btn" onclick={() => onSyncChange(-0.05)} aria-label="Audio earlier">−</button>
				<span class="sync-value" class:sync-negative={audioSync < 0} class:sync-positive={audioSync > 0}>
					{audioSync >= 0 ? '+' : ''}{(audioSync * 1000).toFixed(0)}ms
				</span>
				<button class="sync-btn" onclick={() => onSyncChange(0.05)} aria-label="Audio later">+</button>
				{#if audioSync !== 0}
					<button class="sync-reset" onclick={onSyncReset} aria-label="Reset audio sync">Reset</button>
				{/if}
			</div>
			<div class="sync-hint">− earlier / + later</div>
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

	.track-dropdown {
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

	.track-item {
		display: flex;
		align-items: center;
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

	.track-item:hover {
		background: rgba(255, 255, 255, 0.06);
	}

	.track-item.active {
		color: #E8A849;
	}

	.track-item-check {
		width: 16px;
		color: #E8A849;
	}

	.track-item-label {
		flex: 1;
	}

	.audio-sync-control {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
	}

	.sync-btn {
		width: 32px;
		height: 32px;
		border-radius: 8px;
		background: rgba(255, 255, 255, 0.1);
		border: 1px solid rgba(255, 255, 255, 0.1);
		color: #F5E6C8;
		font-size: 18px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.sync-btn:hover {
		background: rgba(255, 255, 255, 0.2);
	}

	.sync-value {
		min-width: 70px;
		text-align: center;
		font-size: 14px;
		font-weight: 500;
		color: #F5E6C8;
		font-family: monospace;
	}

	.sync-negative {
		color: #4ade80;
	}

	.sync-positive {
		color: #f97316;
	}

	.sync-reset {
		padding: 4px 10px;
		border-radius: 6px;
		background: rgba(255, 255, 255, 0.1);
		border: none;
		color: rgba(245, 230, 200, 0.5);
		font-size: 12px;
		cursor: pointer;
		transition: all 0.2s;
	}

	.sync-reset:hover {
		background: rgba(255, 255, 255, 0.2);
		color: #F5E6C8;
	}

	.sync-hint {
		padding: 4px 16px 8px;
		font-size: 11px;
		color: rgba(245, 230, 200, 0.5);
		text-align: center;
	}
</style>
