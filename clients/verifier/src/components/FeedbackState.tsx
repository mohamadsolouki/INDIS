interface FeedbackStateProps {
  kind: 'loading' | 'empty' | 'error'
  title: string
  message: string
}

const iconByKind: Record<FeedbackStateProps['kind'], string> = {
  loading: '⏳',
  empty: '🗂️',
  error: '⚠️',
}

export default function FeedbackState({ kind, title, message }: FeedbackStateProps) {
  return (
    <section className={`verifier-feedback verifier-feedback--${kind}`} aria-live="polite">
      <div aria-hidden="true">{iconByKind[kind]}</div>
      <div>
        <h2 className="verifier-feedback__title">{title}</h2>
        <p className="verifier-feedback__message">{message}</p>
      </div>
    </section>
  )
}
