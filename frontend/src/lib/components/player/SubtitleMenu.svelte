<script lang="ts">
	import type { SubtitleTrack } from '$lib/api';

	interface Props {
		tracks: SubtitleTrack[];
		selectedIndex: number | null;
		offset: number;
		open: boolean;
		onToggle: () => void;
		onSelect: (index: number | null) => void;
		onOffsetChange: (delta: number) => void;
		onOffsetReset: () => void;
	}

	let { tracks, selectedIndex, offset, open, onToggle, onSelect, onOffsetChange, onOffsetReset }: Props = $props();

	function getLabel(track: SubtitleTrack): string {
		const parts: string[] = [];
		if (track.title) {
			parts.push(track.title);
		} else if (track.language) {
			const langNames: Record<string, string> = {
				eng: 'English', spa: 'Spanish', fre: 'French', fra: 'French',
				deu: 'German', ger: 'German', ita: 'Italian', por: 'Portuguese',
				jpn: 'Japanese', kor: 'Korean', zho: 'Chinese', chi: 'Chinese',
				rus: 'Russian', ara: 'Arabic', hin: 'Hindi', dut: 'Dutch',
				pol: 'Polish', swe: 'Swedish', dan: 'Danish', fin: 'Finnish',
				nor: 'Norwegian', cze: 'Czech', hun: 'Hungarian', gre: 'Greek',
				heb: 'Hebrew', tha: 'Thai', tur: 'Turkish', vie: 'Vietnamese',
				ind: 'Indonesian', und: 'Unknown'
			};
			parts.push(langNames[track.language] || track.language.toUpperCase());
		} else {
			parts.push(`Track ${track.index + 1}`);
		}
		if (track.forced) parts.push('(Forced)');
		if (track.external) parts.push('(External)');
		return parts.join(' ');
	}

	function formatOffset(value: number): string {
		if (value === 0) return '0s';
		const sign = value > 0 ? '+' : '';
		return `${sign}${value.toFixed(1)}s`;
	}
</script>

{#if tracks && tracks.length > 0}
	<div class="relative subtitle-menu">
		<button
			class="player-btn {selectedIndex !== null ? 'active' : ''}"
			onclick={onToggle}
			aria-label="Subtitles"
		>
			<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
				<path d="M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z"/>
			</svg>
		</button>

		{#if open}
			<div class="track-dropdown">
				<button
					class="track-item {selectedIndex === null ? 'active' : ''}"
					onclick={() => onSelect(null)}
				>
					<span class="track-item-check">
						{#if selectedIndex === null}
							<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
								<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
							</svg>
						{/if}
					</span>
					<span class="track-item-label">Off</span>
				</button>
				{#each tracks as track (track.index)}
					<button
						class="track-item {selectedIndex === track.index ? 'active' : ''}"
						onclick={() => onSelect(track.index)}
					>
						<span class="track-item-check">
							{#if selectedIndex === track.index}
								<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
									<path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
								</svg>
							{/if}
						</span>
						<span class="track-item-label">{getLabel(track)}</span>
					</button>
				{/each}

				{#if selectedIndex !== null}
					<div class="timing-divider"></div>
					<div class="timing-header">Timing Offset</div>
					<div class="timing-controls">
						<button class="timing-btn" onclick={() => onOffsetChange(-0.5)} title="Delay subtitles">
							-0.5s
						</button>
						<span class="timing-value">{formatOffset(offset)}</span>
						<button class="timing-btn" onclick={() => onOffsetChange(0.5)} title="Earlier subtitles">
							+0.5s
						</button>
						<button class="timing-btn timing-reset" onclick={onOffsetReset} title="Reset">
							Reset
						</button>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}

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

	.player-btn.active {
		color: #E8A849;
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

	.timing-divider {
		height: 1px;
		background: rgba(245, 230, 200, 0.15);
		margin: 8px 0;
	}

	.timing-header {
		padding: 6px 16px;
		color: rgba(245, 230, 200, 0.5);
		font-size: 11px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.timing-controls {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
	}

	.timing-btn {
		background: rgba(255, 255, 255, 0.06);
		border: 1px solid rgba(245, 230, 200, 0.2);
		color: #F5E6C8;
		padding: 4px 10px;
		border-radius: 6px;
		font-size: 12px;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
	}

	.timing-btn:hover {
		background: rgba(255, 255, 255, 0.1);
		border-color: rgba(245, 230, 200, 0.3);
	}

	.timing-value {
		color: #E8A849;
		font-size: 13px;
		font-weight: 600;
		font-variant-numeric: tabular-nums;
		min-width: 50px;
		text-align: center;
	}

	.timing-reset {
		margin-left: auto;
		color: rgba(245, 230, 200, 0.5);
	}

	.timing-reset:hover {
		color: #F5E6C8;
	}

	.track-dropdown::-webkit-scrollbar {
		width: 6px;
	}

	.track-dropdown::-webkit-scrollbar-track {
		background: transparent;
	}

	.track-dropdown::-webkit-scrollbar-thumb {
		background: rgba(245, 230, 200, 0.2);
		border-radius: 3px;
	}
</style>
