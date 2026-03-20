import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  DocumentTextIcon,
  FingerPrintIcon,
  UserGroupIcon,
  CheckCircleIcon,
  ArrowRightIcon,
  ArrowLeftIcon,
  IdentificationIcon,
} from '@heroicons/react/24/outline';
import { cn } from '../../lib/cn';
import { useEnrollment } from '../../hooks/useEnrollment';
import CameraCapture from '../../components/CameraCapture/CameraCapture';
import type { EnrollmentPathway } from '../../types';

/** Wizard step index — 0-4 inclusive */
type Step = 0 | 1 | 2 | 3 | 4;

/** Enrollment pathway card configuration */
const PATHWAYS: {
  id: EnrollmentPathway;
  icon: React.ElementType;
  colorClass: string;
}[] = [
  {
    id: 'standard',
    icon: DocumentTextIcon,
    colorClass: 'border-blue-200 bg-blue-50 text-blue-700',
  },
  {
    id: 'enhanced',
    icon: IdentificationIcon,
    colorClass: 'border-green-200 bg-green-50 text-green-700',
  },
  {
    id: 'social',
    icon: UserGroupIcon,
    colorClass: 'border-purple-200 bg-purple-50 text-purple-700',
  },
];

/** i18n key for each step's label shown in the progress bar */
const STEP_KEYS: string[] = [
  'enrollment.select_pathway',
  'enrollment.document_capture',
  'enrollment.biometric_capture',
  'enrollment.review',
  'enrollment.success',
];

/**
 * Enrollment — multi-step identity enrollment wizard.
 *
 * Step 0: Select pathway (Standard / Enhanced / Social Attestation)
 * Step 1: Document capture (camera / file upload placeholder)
 * Step 2: Biometric capture (face + fingerprint placeholders)
 * Step 3: Review & submit summary
 * Step 4: Success screen displaying the new DID
 *
 * RTL-first; progress indicator advances left-to-right in LTR and
 * right-to-left in RTL via Tailwind direction utilities.
 */
export default function Enrollment() {
  const { t } = useTranslation();
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const navigate = useNavigate();
  const [step, setStep] = useState<Step>(0);
  const [pathway, setPathway] = useState<EnrollmentPathway | null>(null);
  const [completedDid, setCompletedDid] = useState<string | null>(null);
  const [docImageB64, setDocImageB64] = useState<string | null>(null);
  const [faceImageB64, setFaceImageB64] = useState<string | null>(null);
  const { initiate, submitBiometrics, complete, loading, error } = useEnrollment();

  // ── Step handlers ──────────────────────────────────────────────────────────

  const handleSelectPathway = async (p: EnrollmentPathway) => {
    setPathway(p);
    const result = await initiate(p);
    if (result) setStep(1);
  };

  const handleNext = async () => {
    if (step === 2) {
      // Submit placeholder biometrics; real device capture happens here.
      // Pass captured image data URLs as biometric descriptors.
      // The AI service extracts face embeddings server-side; raw frames are not stored.
      await submitBiometrics({
        face_descriptor: faceImageB64 ?? 'placeholder_face_data',
      });
      setStep(3);
    } else if (step === 3) {
      const result = await complete();
      if (result) {
        setCompletedDid(result.did);
        setStep(4);
      }
    } else {
      setStep((s) => (s < 4 ? ((s + 1) as Step) : s));
    }
  };

  const handleBack = () => {
    if (step > 0 && step < 4) setStep((s) => ((s - 1) as Step));
  };

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div className="min-h-screen bg-gray-50" dir="rtl">
      {/* ── Header ─────────────────────────────────────────────────────────── */}
      <header className="bg-indis-primary px-4 py-4">
        <div className="max-w-lg mx-auto flex items-center gap-3">
          <Link
            to="/"
            className="text-white/80 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-white rounded"
            aria-label={t('enrollment.back_home')}
          >
            {/* In RTL the "back" arrow points to the right (→); in LTR it points left (←) */}
            <ArrowRightIcon className="w-5 h-5 rtl:rotate-0 ltr:rotate-180" aria-hidden="true" />
          </Link>
          <h1 className="text-white font-bold">{t('enrollment.title')}</h1>
        </div>
      </header>

      {/* ── Progress indicator (steps 0–3 only) ────────────────────────────── */}
      {step < 4 && (
        <div className="bg-white border-b border-gray-100" aria-label={t('enrollment.progress')}>
          <div className="max-w-lg mx-auto px-4 py-3">
            <div className="flex gap-1" role="progressbar" aria-valuenow={step} aria-valuemin={0} aria-valuemax={3}>
              {[0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className={cn(
                    'flex-1 h-1.5 rounded-full transition-colors duration-300',
                    i <= step ? 'bg-indis-primary' : 'bg-gray-200',
                  )}
                />
              ))}
            </div>
            <p className="text-xs text-gray-500 mt-2">{t(STEP_KEYS[step])}</p>
          </div>
        </div>
      )}

      {/* ── Page content ───────────────────────────────────────────────────── */}
      <main className="max-w-lg mx-auto px-4 py-6">
        {/* Error banner */}
        {error && (
          <div
            className="mb-4 bg-red-50 border border-red-200 rounded-lg px-4 py-3"
            role="alert"
          >
            <p className="text-red-700 text-sm">{error}</p>
          </div>
        )}

        {/* ── Step 0: Select Pathway ────────────────────────────────────────── */}
        {step === 0 && (
          <section className="space-y-4 animate-fade-in" aria-labelledby="step0-heading">
            <h2 id="step0-heading" className="font-bold text-gray-900">
              {t('enrollment.select_pathway')}
            </h2>
            <p className="text-gray-600 text-sm">{t('enrollment.select_pathway_desc')}</p>
            <div className="space-y-3">
              {PATHWAYS.map(({ id, icon: Icon, colorClass }) => (
                <button
                  key={id}
                  type="button"
                  onClick={() => void handleSelectPathway(id)}
                  disabled={loading}
                  className={cn(
                    'w-full flex items-start gap-4 p-4 rounded-xl border-2 text-right rtl:text-right ltr:text-left transition-all hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary',
                    colorClass,
                    loading && 'opacity-50 cursor-not-allowed',
                  )}
                  aria-busy={loading}
                >
                  <Icon className="w-8 h-8 flex-shrink-0 mt-0.5" aria-hidden="true" />
                  <div>
                    <p className="font-bold">{t(`enrollment.pathway_${id}`)}</p>
                    <p className="text-sm opacity-80 mt-0.5">
                      {t(`enrollment.pathway_${id}_desc`)}
                    </p>
                  </div>
                </button>
              ))}
            </div>
          </section>
        )}

        {/* ── Step 1: Document Capture ──────────────────────────────────────── */}
        {step === 1 && (
          <section className="space-y-4 animate-fade-in" aria-labelledby="step1-heading">
            <h2 id="step1-heading" className="font-bold text-gray-900">
              {t('enrollment.document_capture')}
            </h2>

            {/* Camera capture — falls back to file input if camera not available */}
            {docImageB64 ? (
              <div className="space-y-3">
                <div className="relative rounded-2xl overflow-hidden aspect-video bg-black">
                  <img src={docImageB64} alt="سند گرفته‌شده" className="w-full h-full object-cover" />
                </div>
                <button
                  type="button"
                  onClick={() => setDocImageB64(null)}
                  className="text-indis-primary text-sm flex items-center gap-1 hover:underline"
                >
                  <DocumentTextIcon className="w-4 h-4" /> دوباره اسکن کنید
                </button>
              </div>
            ) : (
              <CameraCapture
                facingMode="environment"
                label={
                  pathway === 'standard' ? 'کارت ملی یا شناسنامه را اسکن کنید' :
                  pathway === 'enhanced' ? 'مدرک ثبت احوال را اسکن کنید' :
                  'مدرک هویتی پایه را اسکن کنید'
                }
                hint="سند را در کادر قرار دهید سپس عکس بگیرید"
                onCapture={(dataUrl) => setDocImageB64(dataUrl)}
              />
            )}

            <NavigationButtons
              onNext={() => void handleNext()}
              onBack={handleBack}
              loading={loading}
              t={t}
              nextLabel={docImageB64 ? undefined : t('enrollment.next')}
            />
          </section>
        )}

        {/* ── Step 2: Biometric Capture ─────────────────────────────────────── */}
        {step === 2 && (
          <section className="space-y-4 animate-fade-in" aria-labelledby="step2-heading">
            <h2 id="step2-heading" className="font-bold text-gray-900">
              {t('enrollment.biometric_capture')}
            </h2>

            <div className="space-y-4">
              {/* Face capture — real camera via getUserMedia */}
              {faceImageB64 ? (
                <div className="space-y-2">
                  <p className="text-sm font-medium text-gray-700" lang="fa">تشخیص چهره</p>
                  <div className="relative rounded-2xl overflow-hidden aspect-square max-w-xs mx-auto bg-black">
                    <img src={faceImageB64} alt="تصویر چهره" className="w-full h-full object-cover" />
                    <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                      {/* Oval face guide */}
                      <div className="w-40 h-52 border-2 border-green-400/70 rounded-full" />
                    </div>
                  </div>
                  <button
                    type="button"
                    onClick={() => setFaceImageB64(null)}
                    className="text-indis-primary text-sm flex items-center gap-1 hover:underline"
                  >
                    دوباره عکس بگیرید
                  </button>
                </div>
              ) : (
                <CameraCapture
                  facingMode="user"
                  label="تشخیص چهره"
                  hint="مستقیم به دوربین نگاه کنید — صورت خود را در کادر بیضی قرار دهید"
                  onCapture={(dataUrl) => setFaceImageB64(dataUrl)}
                />
              )}

              {/* Fingerprint — hardware sensor placeholder; not available in browser */}
              <div
                className="border border-indigo-200 rounded-xl p-4 text-center bg-indigo-50 space-y-1"
                role="note"
                aria-label="اثر انگشت"
              >
                <FingerPrintIcon className="w-8 h-8 text-indigo-400 mx-auto" aria-hidden="true" />
                <p className="font-medium text-indigo-800 text-sm" lang="fa">اثر انگشت</p>
                <p className="text-indigo-500 text-xs" lang="fa">
                  در نسخه موبایل یا دستگاه‌های دارای سنسور فعال می‌شود
                </p>
              </div>
            </div>

            <NavigationButtons
              onNext={() => void handleNext()}
              onBack={handleBack}
              loading={loading}
              t={t}
            />
          </section>
        )}

        {/* ── Step 3: Review & Submit ───────────────────────────────────────── */}
        {step === 3 && (
          <section className="space-y-4 animate-fade-in" aria-labelledby="step3-heading">
            <h2 id="step3-heading" className="font-bold text-gray-900">
              {t('enrollment.review')}
            </h2>

            <dl className="bg-white rounded-xl border border-gray-200 divide-y divide-gray-100">
              <div className="px-4 py-3 flex justify-between text-sm">
                <dt className="text-gray-500" lang="fa">مسیر ثبت‌نام</dt>
                <dd className="font-medium">
                  {pathway ? t(`enrollment.pathway_${pathway}`) : '—'}
                </dd>
              </div>
              <div className="px-4 py-3 flex justify-between text-sm">
                <dt className="text-gray-500" lang="fa">مدارک</dt>
                <dd className="font-medium text-green-600" lang="fa">✓ بارگذاری شد</dd>
              </div>
              <div className="px-4 py-3 flex justify-between text-sm">
                <dt className="text-gray-500" lang="fa">بیومتریک</dt>
                <dd className="font-medium text-green-600" lang="fa">✓ ثبت شد</dd>
              </div>
            </dl>

            <p className="text-gray-500 text-xs" lang="fa">
              با ارسال درخواست، موافقت خود با شرایط استفاده از سامانه هویت دیجیتال ملی را اعلام
              می‌کنید.
            </p>

            <NavigationButtons
              onNext={() => void handleNext()}
              onBack={handleBack}
              loading={loading}
              t={t}
              nextLabel={t('enrollment.submit')}
            />
          </section>
        )}

        {/* ── Step 4: Success ───────────────────────────────────────────────── */}
        {step === 4 && (
          <section
            className="text-center space-y-6 py-8 animate-slide-up"
            aria-labelledby="step4-heading"
            aria-live="polite"
          >
            <CheckCircleIcon
              className="w-20 h-20 text-green-500 mx-auto"
              aria-hidden="true"
            />

            <div>
              <h2 id="step4-heading" className="text-2xl font-bold text-gray-900">
                {t('enrollment.success')}
              </h2>
              <p className="text-gray-500 text-sm mt-2" lang="fa">
                هویت دیجیتال شما با موفقیت ایجاد شد
              </p>
            </div>

            {completedDid && (
              <div className="bg-gray-50 rounded-xl p-3">
                <p className="text-xs text-gray-400 mb-1" lang="fa">DID شما</p>
                <p
                  className="font-mono text-xs text-gray-700 break-all"
                  dir="ltr"
                  aria-label={`DID: ${completedDid}`}
                >
                  {completedDid}
                </p>
              </div>
            )}

            <Link
              to="/"
              className="inline-block bg-indis-primary text-white rounded-xl px-8 py-3 font-medium hover:bg-indis-primary-dark transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary"
              lang="fa"
            >
              بازگشت به خانه
            </Link>
          </section>
        )}
      </main>
    </div>
  );
}

// ── Shared navigation buttons ─────────────────────────────────────────────────

interface NavButtonsProps {
  onNext: () => void;
  onBack: () => void;
  loading: boolean;
  /** Pass `t` directly so this sub-component stays self-contained. */
  t: (key: string) => string;
  nextLabel?: string;
}

/**
 * NavigationButtons — back / next button pair shared across wizard steps.
 *
 * The back arrow points right (→) in RTL and left (←) in LTR.
 * The next arrow is the inverse.
 */
function NavigationButtons({ onNext, onBack, loading, t, nextLabel }: NavButtonsProps) {
  return (
    <div className="flex gap-3 pt-2">
      {/* Back */}
      <button
        type="button"
        onClick={onBack}
        className="flex items-center gap-1 text-gray-600 rounded-xl px-4 py-3 border border-gray-200 hover:bg-gray-50 transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-gray-400"
      >
        <ArrowRightIcon className="w-4 h-4 rtl:rotate-0 ltr:rotate-180" aria-hidden="true" />
        {t('enrollment.back')}
      </button>

      {/* Next / Submit */}
      <button
        type="button"
        onClick={onNext}
        disabled={loading}
        className={cn(
          'flex-1 flex items-center justify-center gap-2 bg-indis-primary text-white rounded-xl px-4 py-3 font-medium hover:bg-indis-primary-dark transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-indis-primary',
          loading && 'opacity-60 cursor-not-allowed',
        )}
        aria-busy={loading}
      >
        {loading ? (
          <span
            className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"
            role="status"
            aria-label={t('common.loading')}
          />
        ) : (
          <ArrowLeftIcon className="w-4 h-4 rtl:rotate-180 ltr:rotate-0" aria-hidden="true" />
        )}
        {nextLabel ?? t('enrollment.next')}
      </button>
    </div>
  );
}
