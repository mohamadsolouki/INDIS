import { useEffect } from 'react';
import {
  CameraIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  XMarkIcon,
} from '@heroicons/react/24/outline';
import { useCamera } from '../../hooks/useCamera';
import { cn } from '../../lib/cn';

export interface CameraCaptureProps {
  /** 'environment' for document scan (rear camera), 'user' for face/biometric. */
  facingMode?: 'user' | 'environment';
  label?: string;
  hint?: string;
  /** Called with base64 JPEG data URL when the user accepts the snapshot. */
  onCapture: (dataUrl: string) => void;
  className?: string;
}

/**
 * CameraCapture — live viewfinder with snapshot and retake support.
 *
 * Used in the enrollment wizard:
 *  - Document capture (step 1): rear camera, document framing hint
 *  - Biometric capture (step 2): front camera, face framing hint
 *
 * The component automatically starts the camera on mount and stops the stream
 * on unmount (handled by `useCamera`).
 */
export default function CameraCapture({
  facingMode = 'environment',
  label,
  hint,
  onCapture,
  className,
}: CameraCaptureProps) {
  const {
    videoRef,
    canvasRef,
    isStreaming,
    capture,
    error,
    startCamera,
    stopCamera,
    takeSnapshot,
    reset,
  } = useCamera();

  useEffect(() => {
    void startCamera(facingMode);
    return () => stopCamera();
    // Only run on mount / facingMode change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [facingMode]);

  function handleAccept() {
    if (capture?.imageDataUrl) {
      onCapture(capture.imageDataUrl);
      stopCamera();
    }
  }

  return (
    <div className={cn('flex flex-col gap-3', className)}>
      {label && <p className="font-medium text-gray-800 text-sm">{label}</p>}

      {/* Error state */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-xl p-4 text-center space-y-3">
          <p className="text-red-700 text-sm">{error}</p>
          <button
            type="button"
            onClick={() => { reset(); void startCamera(facingMode); }}
            className="text-red-600 text-xs flex items-center gap-1 mx-auto hover:underline"
          >
            <ArrowPathIcon className="w-3.5 h-3.5" />
            تلاش مجدد
          </button>
        </div>
      )}

      {/* Preview / viewfinder */}
      {!error && !capture && (
        <div className="relative bg-black rounded-2xl overflow-hidden aspect-video">
          <video
            ref={videoRef}
            autoPlay
            playsInline
            muted
            className="w-full h-full object-cover"
          />

          {/* Frame overlay */}
          {isStreaming && (
            <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
              <div className="w-3/4 h-3/4 border-2 border-white/60 rounded-xl" />
            </div>
          )}

          {/* Loading overlay */}
          {!isStreaming && (
            <div className="absolute inset-0 flex items-center justify-center bg-black/40">
              <div className="w-8 h-8 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            </div>
          )}
        </div>
      )}

      {/* Snapshot preview */}
      {capture?.imageDataUrl && (
        <div className="relative rounded-2xl overflow-hidden aspect-video bg-black">
          <img
            src={capture.imageDataUrl}
            alt="تصویر گرفته‌شده"
            className="w-full h-full object-cover"
          />
        </div>
      )}

      {/* Hidden canvas used by takeSnapshot */}
      <canvas ref={canvasRef} className="hidden" />

      {/* Hint */}
      {hint && !capture && (
        <p className="text-center text-gray-500 text-xs">{hint}</p>
      )}

      {/* Action buttons */}
      <div className="flex gap-3">
        {!capture ? (
          <button
            type="button"
            onClick={() => takeSnapshot()}
            disabled={!isStreaming}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 bg-indis-primary text-white rounded-2xl py-3 font-medium hover:bg-indis-primary-dark transition-colors',
              !isStreaming && 'opacity-50 cursor-not-allowed',
            )}
          >
            <CameraIcon className="w-5 h-5" />
            عکس بگیر
          </button>
        ) : (
          <>
            {/* Retake */}
            <button
              type="button"
              onClick={() => { reset(); void startCamera(facingMode); }}
              className="flex items-center gap-2 border border-gray-200 text-gray-600 rounded-2xl px-4 py-3 hover:bg-gray-50 transition-colors"
            >
              <XMarkIcon className="w-4 h-4" />
              دوباره
            </button>

            {/* Accept */}
            <button
              type="button"
              onClick={handleAccept}
              className="flex-1 flex items-center justify-center gap-2 bg-green-600 text-white rounded-2xl py-3 font-medium hover:bg-green-700 transition-colors"
            >
              <CheckCircleIcon className="w-5 h-5" />
              تأیید
            </button>
          </>
        )}
      </div>
    </div>
  );
}
