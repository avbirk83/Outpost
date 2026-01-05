<script lang="ts">
	interface Props {
		seasonNumber: number;
		episodeNumber: number;
		title: string;
		countdown: number;
		onCancel: () => void;
		onPlay: () => void;
	}

	let { seasonNumber, episodeNumber, title, countdown, onCancel, onPlay }: Props = $props();
</script>

<div class="next-episode-popup">
	<div class="next-episode-content">
		<div class="next-episode-label">Up Next</div>
		<div class="next-episode-title">S{seasonNumber}E{episodeNumber}: {title || 'Next Episode'}</div>
		<div class="next-episode-actions">
			<button class="next-episode-btn cancel" onclick={onCancel}>
				Cancel
			</button>
			<button class="next-episode-btn play" onclick={onPlay}>
				Play Now ({countdown}s)
			</button>
		</div>
	</div>
</div>

<style>
	.next-episode-popup {
		position: absolute;
		bottom: 120px;
		right: 24px;
		background: rgba(0, 0, 0, 0.9);
		backdrop-filter: blur(16px);
		border: 1px solid rgba(245, 230, 200, 0.2);
		border-radius: 16px;
		padding: 20px 24px;
		z-index: 45;
		min-width: 300px;
		box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
		animation: slideIn 0.3s ease-out;
	}

	@keyframes slideIn {
		from {
			opacity: 0;
			transform: translateX(20px);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}

	.next-episode-content {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.next-episode-label {
		color: rgba(245, 230, 200, 0.5);
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 1px;
	}

	.next-episode-title {
		color: #F5E6C8;
		font-size: 16px;
		font-weight: 600;
		line-height: 1.3;
	}

	.next-episode-actions {
		display: flex;
		gap: 12px;
		margin-top: 4px;
	}

	.next-episode-btn {
		flex: 1;
		padding: 12px 16px;
		border-radius: 10px;
		font-size: 14px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
		border: none;
	}

	.next-episode-btn.cancel {
		background: rgba(255, 255, 255, 0.1);
		color: #F5E6C8;
		border: 1px solid rgba(245, 230, 200, 0.2);
	}

	.next-episode-btn.cancel:hover {
		background: rgba(255, 255, 255, 0.2);
	}

	.next-episode-btn.play {
		background: #E8A849;
		color: #0a0a0a;
	}

	.next-episode-btn.play:hover {
		background: #f0b85d;
	}
</style>
