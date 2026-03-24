export type VerificationStatus = 'approved' | 'denied'

interface VerificationPresentation {
  labelFa: string
  labelEn: string
  tone: 'ok' | 'fail'
}

const presentationMap: Record<VerificationStatus, VerificationPresentation> = {
  approved: {
    labelFa: 'تأیید شد',
    labelEn: 'APPROVED',
    tone: 'ok',
  },
  denied: {
    labelFa: 'رد شد',
    labelEn: 'DENIED',
    tone: 'fail',
  },
}

export function verificationStatusFromBoolean(valid: boolean): VerificationStatus {
  return valid ? 'approved' : 'denied'
}

export function verificationStatusPresentation(status: VerificationStatus): VerificationPresentation {
  return presentationMap[status]
}
