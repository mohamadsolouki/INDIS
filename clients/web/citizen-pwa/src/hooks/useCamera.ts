import { useRef, useState, useCallback, useEffect } from 'react';

export interface CameraCapture {
  /** Base64-encoded JPEG data URL of the captured frame. */
  imageDataUrl: string | null;
}

export interface UseCameraReturn {
  videoRef: React.RefObject<HTMLVideoElement>;
  canvasRef: React.RefObject<HTMLCanvasElement>;
  isStreaming: boolean;
  capture: CameraCapture | null;
  error: string | null;
  startCamera: (facingMode?: 'user' | 'environment') => Promise<void>;
  stopCamera: () => void;
  takeSnapshot: () => string | null;
  reset: () => void;
}

/**
 * useCamera — React hook for accessing the device camera via `getUserMedia`.
 *
 * Used in the enrollment wizard for document and biometric capture.
 * PRD FR-002: document images are sent to the AI service for OCR.
 * PRD FR-003: face descriptors are extracted server-side; raw frames are never stored.
 *
 * The hook attaches the media stream to `videoRef` which the caller renders
 * as `<video ref={videoRef} autoPlay playsInline muted />`.
 * Call `takeSnapshot()` to capture a frame from the live feed — it draws to
 * `canvasRef` and returns a base64 JPEG data URL.
 */
export function useCamera(): UseCameraReturn {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const [isStreaming, setIsStreaming] = useState(false);
  const [capture, setCapture] = useState<CameraCapture | null>(null);
  const [error, setError] = useState<string | null>(null);

  const stopCamera = useCallback(() => {
    streamRef.current?.getTracks().forEach((t) => t.stop());
    streamRef.current = null;
    if (videoRef.current) videoRef.current.srcObject = null;
    setIsStreaming(false);
  }, []);

  // Stop the stream when the component unmounts.
  useEffect(() => () => { stopCamera(); }, [stopCamera]);

  const startCamera = useCallback(async (facingMode: 'user' | 'environment' = 'environment') => {
    setError(null);
    setCapture(null);

    if (!navigator.mediaDevices?.getUserMedia) {
      setError('دسترسی به دوربین در این مرورگر پشتیبانی نمی‌شود');
      return;
    }

    try {
      const constraints: MediaStreamConstraints = {
        video: {
          facingMode,
          width: { ideal: 1280 },
          height: { ideal: 720 },
        },
        audio: false,
      };

      const stream = await navigator.mediaDevices.getUserMedia(constraints);
      streamRef.current = stream;

      if (videoRef.current) {
        videoRef.current.srcObject = stream;
        await videoRef.current.play();
        setIsStreaming(true);
      }
    } catch (err) {
      if (err instanceof DOMException) {
        if (err.name === 'NotAllowedError') {
          setError('دسترسی به دوربین رد شد. لطفاً از تنظیمات مرورگر اجازه دهید.');
        } else if (err.name === 'NotFoundError') {
          setError('دوربینی یافت نشد.');
        } else {
          setError(`خطای دوربین: ${err.message}`);
        }
      } else {
        setError('خطای ناشناخته در دوربین');
      }
    }
  }, []);

  const takeSnapshot = useCallback((): string | null => {
    const video = videoRef.current;
    const canvas = canvasRef.current;
    if (!video || !canvas || !isStreaming) return null;

    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    const ctx = canvas.getContext('2d');
    if (!ctx) return null;

    ctx.drawImage(video, 0, 0);
    const dataUrl = canvas.toDataURL('image/jpeg', 0.85);
    setCapture({ imageDataUrl: dataUrl });
    return dataUrl;
  }, [isStreaming]);

  const reset = useCallback(() => {
    setCapture(null);
    setError(null);
  }, []);

  return { videoRef, canvasRef, isStreaming, capture, error, startCamera, stopCamera, takeSnapshot, reset };
}
