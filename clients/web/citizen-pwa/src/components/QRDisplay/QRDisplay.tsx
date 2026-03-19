import { useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { useTranslation } from 'react-i18next';
import { ArrowsPointingOutIcon, ArrowsPointingInIcon, ArrowDownTrayIcon } from '@heroicons/react/24/outline';
import { cn } from '../../lib/cn';

interface QRDisplayProps {
  value: string;          // The data to encode in the QR
  label?: string;         // Label shown below QR
  size?: number;          // Default 200
  showDownload?: boolean;
  className?: string;
}

export default function QRDisplay({ value, label, size = 200, showDownload = true, className }: QRDisplayProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);

  const qrSize = expanded ? Math.min(window.innerWidth - 64, 320) : size;

  const handleDownload = () => {
    // Find the SVG element and convert to PNG
    const svg = document.querySelector('.indis-qr-svg') as SVGElement | null;
    if (!svg) return;

    const canvas = document.createElement('canvas');
    canvas.width = 512;
    canvas.height = 512;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const svgData = new XMLSerializer().serializeToString(svg);
    const img = new Image();
    img.onload = () => {
      ctx.drawImage(img, 0, 0, 512, 512);
      const a = document.createElement('a');
      a.href = canvas.toDataURL('image/png');
      a.download = 'indis-qr.png';
      a.click();
    };
    img.src = `data:image/svg+xml;base64,${btoa(svgData)}`;
  };

  return (
    <div className={cn('flex flex-col items-center gap-3', className)}>
      {/* QR Code */}
      <div
        className={cn(
          'relative bg-white p-4 rounded-2xl shadow-sm border border-gray-100',
          expanded && 'fixed inset-0 z-50 flex items-center justify-center bg-black/60 rounded-none border-none shadow-none',
        )}
      >
        {expanded && (
          <div className="bg-white p-6 rounded-2xl">
            <QRCodeSVG
              value={value}
              size={qrSize}
              level="H"
              includeMargin
              className="indis-qr-svg"
              imageSettings={{
                src: '/icons/icon-192.svg',
                height: 40,
                width: 40,
                excavate: true,
              }}
            />
          </div>
        )}
        {!expanded && (
          <QRCodeSVG
            value={value}
            size={qrSize}
            level="H"
            includeMargin
            className="indis-qr-svg"
            imageSettings={{
              src: '/icons/icon-192.svg',
              height: 24,
              width: 24,
              excavate: true,
            }}
          />
        )}

        {/* Expand/collapse button */}
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className={cn(
            'absolute top-2 end-2 p-1 rounded-lg bg-gray-100 hover:bg-gray-200 transition-colors',
            expanded && 'top-8 end-8 bg-white/90',
          )}
          aria-label={expanded ? 'کوچک‌تر کردن' : 'بزرگ‌تر کردن'}
        >
          {expanded
            ? <ArrowsPointingInIcon className="w-4 h-4 text-gray-600" />
            : <ArrowsPointingOutIcon className="w-4 h-4 text-gray-600" />
          }
        </button>
      </div>

      {/* Label */}
      {label && (
        <p className="text-xs text-gray-500 text-center">{label}</p>
      )}

      {/* Download button */}
      {showDownload && (
        <button
          type="button"
          onClick={handleDownload}
          className="flex items-center gap-1 text-indis-primary text-xs hover:underline"
        >
          <ArrowDownTrayIcon className="w-3.5 h-3.5" />
          دانلود QR
        </button>
      )}
    </div>
  );
}
