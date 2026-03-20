package org.indis.app.ui.enrollment

import android.Manifest
import android.content.pm.PackageManager
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.ImageFormat
import android.hardware.camera2.CameraCaptureSession
import android.hardware.camera2.CameraCharacteristics
import android.hardware.camera2.CameraDevice
import android.hardware.camera2.CameraManager
import android.media.ImageReader
import android.os.Bundle
import android.os.Handler
import android.os.HandlerThread
import android.util.Base64
import android.view.LayoutInflater
import android.view.Surface
import android.view.TextureView
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.ImageView
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import org.indis.app.R
import java.io.ByteArrayOutputStream

/**
 * Step 1 of enrollment — capture the citizen's identity document.
 *
 * Uses Camera2 API with the rear-facing camera.
 * The captured JPEG is base64-encoded and forwarded to [EnrollmentActivity.onDocumentCaptured].
 *
 * PRD FR-002 (document capture), FR-003 (OCR trigger happens server-side after upload).
 */
class DocumentFragment : Fragment() {

    private lateinit var textureView: TextureView
    private lateinit var imgPreview: ImageView
    private lateinit var btnCapture: Button
    private lateinit var btnRetake: Button

    private lateinit var cameraManager: CameraManager
    private var cameraDevice: CameraDevice? = null
    private var captureSession: CameraCaptureSession? = null
    private lateinit var imageReader: ImageReader
    private lateinit var backgroundHandler: Handler
    private lateinit var backgroundThread: HandlerThread

    private val requestPermission =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
            if (granted) startCamera()
        }

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?,
    ): View = inflater.inflate(R.layout.fragment_document, container, false)

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        textureView = view.findViewById(R.id.texture_camera)
        imgPreview  = view.findViewById(R.id.img_preview)
        btnCapture  = view.findViewById(R.id.btn_capture)
        btnRetake   = view.findViewById(R.id.btn_retake)

        cameraManager = requireContext().getSystemService(CameraManager::class.java)

        btnCapture.setOnClickListener { captureImage() }
        btnRetake.setOnClickListener  { retake() }
    }

    override fun onResume() {
        super.onResume()
        startBackgroundThread()
        if (ContextCompat.checkSelfPermission(requireContext(), Manifest.permission.CAMERA)
            == PackageManager.PERMISSION_GRANTED
        ) {
            startCamera()
        } else {
            requestPermission.launch(Manifest.permission.CAMERA)
        }
    }

    override fun onPause() {
        closeCamera()
        stopBackgroundThread()
        super.onPause()
    }

    private fun startBackgroundThread() {
        backgroundThread = HandlerThread("CameraBackground").also { it.start() }
        backgroundHandler = Handler(backgroundThread.looper)
    }

    private fun stopBackgroundThread() {
        backgroundThread.quitSafely()
        backgroundThread.join()
    }

    private fun startCamera() {
        val cameraId = cameraManager.cameraIdList.firstOrNull { id ->
            cameraManager.getCameraCharacteristics(id)
                .get(CameraCharacteristics.LENS_FACING) == CameraCharacteristics.LENS_FACING_BACK
        } ?: return

        imageReader = ImageReader.newInstance(1280, 960, ImageFormat.JPEG, 1)
        imageReader.setOnImageAvailableListener({ reader ->
            val image = reader.acquireLatestImage() ?: return@setOnImageAvailableListener
            val buffer = image.planes[0].buffer
            val bytes  = ByteArray(buffer.remaining()).also { buffer.get(it) }
            image.close()
            processCapture(bytes)
        }, backgroundHandler)

        @Suppress("MissingPermission")
        cameraManager.openCamera(cameraId, object : CameraDevice.StateCallback() {
            override fun onOpened(camera: CameraDevice) {
                cameraDevice = camera
                startPreview(camera)
            }
            override fun onDisconnected(camera: CameraDevice) { camera.close() }
            override fun onError(camera: CameraDevice, error: Int) { camera.close() }
        }, backgroundHandler)
    }

    private fun startPreview(camera: CameraDevice) {
        val surface      = Surface(textureView.surfaceTexture)
        val readerSurface = imageReader.surface
        val previewRequest = camera.createCaptureRequest(CameraDevice.TEMPLATE_PREVIEW).apply {
            addTarget(surface)
        }.build()

        camera.createCaptureSession(
            listOf(surface, readerSurface),
            object : CameraCaptureSession.StateCallback() {
                override fun onConfigured(session: CameraCaptureSession) {
                    captureSession = session
                    session.setRepeatingRequest(previewRequest, null, backgroundHandler)
                }
                override fun onConfigureFailed(session: CameraCaptureSession) {}
            },
            backgroundHandler,
        )
    }

    private fun captureImage() {
        val camera  = cameraDevice ?: return
        val session = captureSession ?: return

        val captureRequest = camera.createCaptureRequest(CameraDevice.TEMPLATE_STILL_CAPTURE).apply {
            addTarget(imageReader.surface)
        }.build()
        session.capture(captureRequest, null, backgroundHandler)
    }

    private fun processCapture(jpegBytes: ByteArray) {
        val bitmap   = BitmapFactory.decodeByteArray(jpegBytes, 0, jpegBytes.size)
        val b64      = encodeToBase64(bitmap)

        requireActivity().runOnUiThread {
            imgPreview.setImageBitmap(bitmap)
            imgPreview.visibility       = View.VISIBLE
            textureView.visibility      = View.GONE
            btnCapture.visibility       = View.GONE
            btnRetake.visibility        = View.VISIBLE
            (activity as? EnrollmentActivity)?.onDocumentCaptured(b64)
        }
    }

    private fun retake() {
        imgPreview.visibility  = View.GONE
        textureView.visibility = View.VISIBLE
        btnCapture.visibility  = View.VISIBLE
        btnRetake.visibility   = View.GONE
        (activity as? EnrollmentActivity)?.onDocumentCaptured("")
    }

    private fun closeCamera() {
        captureSession?.close(); captureSession = null
        cameraDevice?.close();   cameraDevice   = null
        imageReader.close()
    }

    private fun encodeToBase64(bitmap: Bitmap): String {
        val out = ByteArrayOutputStream()
        bitmap.compress(Bitmap.CompressFormat.JPEG, 85, out)
        return Base64.encodeToString(out.toByteArray(), Base64.NO_WRAP)
    }
}
