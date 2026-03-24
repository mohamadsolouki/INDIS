interface FeedbackStateProps {
  kind: 'loading' | 'empty' | 'error' | 'success'
  title: string
  message: string
}

const iconByKind: Record<FeedbackStateProps['kind'], string> = {
  loading: '⏳',
  empty: '🗂️',
  error: '⚠️',
  success: '✅',
}

export default function FeedbackState({ kind, title, message }: FeedbackStateProps) {
  return (
    <section className={`feedback-panel feedback-panel--${kind}`} aria-live="polite">
      <div className="feedback-icon" aria-hidden="true">{iconByKind[kind]}</div>
      <div>
        <h2 className="feedback-title">{title}</h2>
        <p className="feedback-message">{message}</p>
      </div>
    </section>
  )
}
