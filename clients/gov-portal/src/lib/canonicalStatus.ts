export type CanonicalStatusTone = 'warning' | 'info' | 'success' | 'error' | 'default'

export type EnrollmentStatus = 'pending' | 'under_review' | 'approved' | 'rejected' | 'requires_biometric'
export type IssuanceStatus = 'queued' | 'issuing' | 'issued' | 'failed'

const enrollmentMap: Record<EnrollmentStatus, { label: string; tone: CanonicalStatusTone }> = {
  pending: { label: 'در انتظار', tone: 'warning' },
  under_review: { label: 'در حال بررسی', tone: 'info' },
  approved: { label: 'تأیید شده', tone: 'success' },
  rejected: { label: 'رد شده', tone: 'error' },
  requires_biometric: { label: 'نیاز به بیومتریک', tone: 'info' },
}

const issuanceMap: Record<IssuanceStatus, { label: string; tone: CanonicalStatusTone }> = {
  queued: { label: 'در صف', tone: 'warning' },
  issuing: { label: 'در حال صدور', tone: 'info' },
  issued: { label: 'صادر شد', tone: 'success' },
  failed: { label: 'خطا', tone: 'error' },
}

function toBadgeClass(tone: CanonicalStatusTone): string {
  return `status-badge--${tone}`
}

export function enrollmentStatusLabel(status: EnrollmentStatus): string {
  return enrollmentMap[status]?.label ?? status
}

export function enrollmentStatusBadgeClass(status: EnrollmentStatus): string {
  return toBadgeClass(enrollmentMap[status]?.tone ?? 'default')
}

export function issuanceStatusLabel(status: IssuanceStatus): string {
  return issuanceMap[status]?.label ?? status
}

export function issuanceStatusBadgeClass(status: IssuanceStatus): string {
  return toBadgeClass(issuanceMap[status]?.tone ?? 'default')
}
