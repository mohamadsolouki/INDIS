# INDIS — Mobile Applications

## Android (Kotlin / Jetpack Compose)

> Primary platform — highest Iranian market share

- Minimum version: Android 8.0 (API 26)
- Language: Kotlin
- UI: Jetpack Compose
- ZK proofs: On-device via WASM/native bindings

## iOS (Swift / SwiftUI)

> Secondary platform

- Minimum version: iOS 14.0
- Language: Swift
- UI: SwiftUI
- Secure Enclave for key storage

## HarmonyOS (ArkTS / ArkUI)

> Required for Huawei devices common in Iran

- Minimum version: HarmonyOS 2.0
- Language: ArkTS
- UI: ArkUI

## Common Requirements (PRD §FR-005, §FR-006)

- **Persian-first RTL** design — all interfaces designed RTL first
- **Vazirmatn** typography
- **Solar Hijri** calendar default
- **Persian numerals** (۰۱۲۳۴۵۶۷۸۹) default
- **DID key generation ON-DEVICE** — private key never leaves the device
- **Offline ZK proof generation** — full credential presentation without network
- **Encrypted credential wallet** — AES-256-GCM
