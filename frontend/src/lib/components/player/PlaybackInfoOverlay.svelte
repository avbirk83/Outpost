<script lang="ts">
	export interface PlaybackInfo {
		playMethod: string;
		container: string;
		resolution: string;
		videoCodec: string;
		audioCodec: string;
		audioChannels: string;
		droppedFrames: number;
		totalFrames: number;
		bitrate: string;
	}

	interface Props {
		info: PlaybackInfo;
	}

	let { info }: Props = $props();
</script>

<div class="playback-info-overlay">
	<div class="info-title">Playback Info</div>

	<div class="info-section">
		<div class="info-section-title">Playback</div>
		<div class="info-row" class:good={info.playMethod === 'Direct Play'} class:warning={info.playMethod === 'Transcode'}>
			<span>Play Method</span>
			<span>{info.playMethod}</span>
		</div>
		<div class="info-row">
			<span>Container</span>
			<span>{info.container}</span>
		</div>
		{#if info.bitrate !== 'N/A'}
			<div class="info-row">
				<span>Bitrate</span>
				<span>{info.bitrate}</span>
			</div>
		{/if}
	</div>

	<div class="info-section">
		<div class="info-section-title">Video</div>
		<div class="info-row">
			<span>Resolution</span>
			<span>{info.resolution}</span>
		</div>
		<div class="info-row">
			<span>Codec</span>
			<span>{info.videoCodec}</span>
		</div>
		<div class="info-row" class:good={info.droppedFrames === 0} class:warning={info.droppedFrames > 0 && info.droppedFrames < 10} class:bad={info.droppedFrames >= 10}>
			<span>Dropped Frames</span>
			<span>{info.droppedFrames}{info.totalFrames > 0 ? ` / ${info.totalFrames}` : ''}</span>
		</div>
	</div>

	<div class="info-section">
		<div class="info-section-title">Audio</div>
		<div class="info-row">
			<span>Codec</span>
			<span>{info.audioCodec}{info.audioChannels ? ` (${info.audioChannels})` : ''}</span>
		</div>
	</div>

	<div class="info-hint">Press 'i' to close</div>
</div>

<style>
	.playback-info-overlay {
		position: absolute;
		top: 80px;
		left: 24px;
		background: rgba(0, 0, 0, 0.85);
		backdrop-filter: blur(12px);
		border: 1px solid rgba(245, 230, 200, 0.15);
		border-radius: 12px;
		padding: 16px 20px;
		z-index: 40;
		min-width: 280px;
		font-family: 'SF Mono', 'Menlo', 'Monaco', 'Consolas', monospace;
	}

	.info-title {
		color: #E8A849;
		font-size: 13px;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 1px;
		margin-bottom: 12px;
		padding-bottom: 8px;
		border-bottom: 1px solid rgba(245, 230, 200, 0.15);
	}

	.info-section {
		margin-bottom: 12px;
	}

	.info-section:last-of-type {
		margin-bottom: 0;
	}

	.info-section-title {
		color: rgba(245, 230, 200, 0.5);
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		margin-bottom: 6px;
	}

	.info-row {
		display: flex;
		justify-content: space-between;
		gap: 16px;
		padding: 4px 0;
		font-size: 12px;
	}

	.info-row span:first-child {
		color: rgba(245, 230, 200, 0.5);
	}

	.info-row span:last-child {
		color: #F5E6C8;
		font-weight: 500;
		text-align: right;
	}

	.info-row.good span:last-child {
		color: #4ade80;
	}

	.info-row.warning span:last-child {
		color: #fbbf24;
	}

	.info-row.bad span:last-child {
		color: #f87171;
	}

	.info-hint {
		color: rgba(245, 230, 200, 0.5);
		font-size: 10px;
		margin-top: 12px;
		padding-top: 8px;
		border-top: 1px solid rgba(245, 230, 200, 0.1);
		text-align: center;
	}
</style>
