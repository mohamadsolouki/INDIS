import SwiftUI
import AVFoundation

/// Live QR/barcode scanner using AVCaptureSession.
///
/// Decodes QR codes and calls `onScan` with the raw string payload.
/// Shows a centred 250×250 finder box and torch toggle button.
struct QRScannerView: UIViewControllerRepresentable {

    let onScan: (String) -> Void
    let onError: (String) -> Void

    func makeUIViewController(context: Context) -> QRScannerViewController {
        let vc = QRScannerViewController()
        vc.delegate = context.coordinator
        return vc
    }

    func updateUIViewController(_ uiViewController: QRScannerViewController, context: Context) {}

    func makeCoordinator() -> Coordinator { Coordinator(onScan: onScan, onError: onError) }

    final class Coordinator: NSObject, QRScannerDelegate {
        let onScan: (String) -> Void
        let onError: (String) -> Void
        init(onScan: @escaping (String) -> Void, onError: @escaping (String) -> Void) {
            self.onScan = onScan; self.onError = onError
        }
        func didScan(payload: String) { onScan(payload) }
        func didFail(error: String)   { onError(error) }
    }
}

protocol QRScannerDelegate: AnyObject {
    func didScan(payload: String)
    func didFail(error: String)
}

/// UIKit view controller wrapping AVCaptureSession for QR scanning.
final class QRScannerViewController: UIViewController, AVCaptureMetadataOutputObjectsDelegate {

    weak var delegate: QRScannerDelegate?

    private var captureSession: AVCaptureSession?
    private var previewLayer: AVCaptureVideoPreviewLayer?
    private var scanned = false

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .black
        setupSession()
    }

    override func viewWillAppear(_ animated: Bool) {
        super.viewWillAppear(animated)
        scanned = false
        captureSession?.startRunning()
    }

    override func viewWillDisappear(_ animated: Bool) {
        super.viewWillDisappear(animated)
        captureSession?.stopRunning()
    }

    private func setupSession() {
        let session = AVCaptureSession()
        guard let device = AVCaptureDevice.default(for: .video),
              let input = try? AVCaptureDeviceInput(device: device) else {
            delegate?.didFail(error: "دسترسی به دوربین ممکن نیست")
            return
        }
        session.addInput(input)

        let output = AVCaptureMetadataOutput()
        session.addOutput(output)
        output.setMetadataObjectsDelegate(self, queue: .main)
        output.metadataObjectTypes = [.qr]

        // Constrain to centre viewfinder box
        output.rectOfInterest = CGRect(x: 0.25, y: 0.25, width: 0.5, height: 0.5)

        let preview = AVCaptureVideoPreviewLayer(session: session)
        preview.frame = view.bounds
        preview.videoGravity = .resizeAspectFill
        view.layer.addSublayer(preview)
        previewLayer = preview

        // Finder overlay
        let box = UIView(frame: CGRect(x: (view.bounds.width - 250) / 2,
                                       y: (view.bounds.height - 250) / 2,
                                       width: 250, height: 250))
        box.layer.borderColor = UIColor.systemBlue.cgColor
        box.layer.borderWidth = 2
        box.layer.cornerRadius = 12
        view.addSubview(box)

        captureSession = session
        DispatchQueue.global(qos: .userInitiated).async { session.startRunning() }
    }

    func metadataOutput(_ output: AVCaptureMetadataOutput,
                        didOutput metadataObjects: [AVMetadataObject],
                        from connection: AVCaptureConnection) {
        guard !scanned,
              let obj = metadataObjects.first as? AVMetadataMachineReadableCodeObject,
              obj.type == .qr,
              let payload = obj.stringValue else { return }
        scanned = true
        captureSession?.stopRunning()
        delegate?.didScan(payload: payload)
    }
}
