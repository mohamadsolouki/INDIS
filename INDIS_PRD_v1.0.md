# سند نیازمندی‌های محصول — سیستم هویت دیجیتال ملی ایران
# Product Requirements Document — Iran National Digital Identity System (INDIS)

> **نسخه / Version:** 1.0  
> **تاریخ / Date:** ۱۴۰۵ / 2026  
> **طبقه‌بندی / Classification:** Strategic Planning — Public Draft  
> **زبان اصلی / Primary Language:** فارسی (Persian) — English Co-Primary  
> **وضعیت / Status:** Draft for Stakeholder Review

---

> ⚠️ **یادداشت بلاکچین / Blockchain Notice**  
> The blockchain layer has **not yet been selected**. This document treats the distributed ledger as a **pluggable component**. All architecture is designed to support **Hyperledger Fabric** as the primary candidate, with requirements written to avoid vendor lock-in. The final selection should follow a formal technical evaluation. See [Section 5.3](#53-blockchain-abstraction-layer) for details.

---

## فهرست مطالب / Table of Contents

1. [خلاصه اجرایی / Executive Summary](#part-i-executive-summary-and-vision)
2. [تحلیل ذینفعان / Stakeholder Analysis](#part-ii-stakeholder-analysis)
3. [نیازمندی‌های کارکردی / Functional Requirements](#part-iii-functional-requirements)
4. [نیازمندی‌های غیرکارکردی / Non-Functional Requirements](#part-iv-non-functional-requirements)
5. [معماری سیستم / System Architecture](#part-v-system-architecture)
6. [پشته فناوری / Technology Stack](#part-vi-technology-stack)
7. [نقشه راه پیاده‌سازی / Implementation Roadmap](#part-vii-implementation-roadmap)
8. [چارچوب بلاکچین / Blockchain Framework](#part-viii-blockchain-framework)
9. [سؤالات باز / Open Questions](#part-ix-open-questions-and-decisions-required)

---

# PART I: Executive Summary and Vision
# بخش اول: خلاصه اجرایی و چشم‌انداز

## 1.1 Document Purpose / هدف سند

This Product Requirements Document defines the complete technical, functional, and non-functional requirements for the **Iran National Digital Identity System (INDIS)** — a sovereign digital identity infrastructure designed to serve as the foundational trust layer for post-transition Iran.

این سند نیازمندی‌های کامل فنی، کارکردی و غیرکارکردی **سیستم هویت دیجیتال ملی ایران (INDIS)** را تعریف می‌کند — زیرساختی برای هویت دیجیتال ملی‌حاکمیتی که به‌عنوان لایه اعتماد بنیادی برای ایران پس از گذار طراحی شده است.

**Intended Audience / مخاطبان هدف:**
- Technical architects and software engineers / معماران فناوری و مهندسان نرم‌افزار
- Government policy stakeholders / ذینفعان سیاست‌گذاری دولتی
- International development and audit partners / شرکای توسعه و ممیزی بین‌المللی
- Security auditors and privacy researchers / ممیزان امنیتی و پژوهشگران حریم خصوصی

---

## 1.2 Vision Statement / بیانیه چشم‌انداز

INDIS will be the most **privacy-respecting**, **cryptographically verifiable**, and **institutionally trustworthy** national identity system in the Middle East and Central Asia — exceeding the UAE Pass model across three dimensions:

| Dimension | UAE Pass | INDIS Target |
|-----------|----------|--------------|
| Privacy Architecture | Centralised data store | ZK-proof selective disclosure; citizen-held keys |
| Trust Model | Government-centralised | Blockchain-anchored, decentralised verification |
| Inclusivity | Urban smartphone-first | Offline-capable, multilingual, social attestation pathway |
| Language Support | Arabic + English | **Persian-first RTL**, 8 languages, minority dialect support |
| Post-Quantum Security | Not addressed | NIST PQC standards for long-term credentials |

---

## 1.3 Strategic Objectives / اهداف راهبردی

The system must achieve the following outcomes within its first **180 days** of phased deployment, aligned directly with the Emergency Phase Booklet:

| Objective | Booklet Alignment | Success Metric |
|-----------|-------------------|----------------|
| **Referendum Readiness** | Political Chapter — referendum within 4 months | Verified voter roll operational before referendum date |
| **Pension Ghost Elimination** | Macroeconomic Chapter — fiscal stabilisation | Ghost beneficiary rate reduced by ≥80% within 90 days |
| **Personnel Vetting** | Military Chapter — vetting by Day 40 | 100% of priority personnel credentialed by Day 40 |
| **Justice Infrastructure** | Transitional Justice — Truth Commission | Anonymous testimony system operational by Day 60 |
| **Service Continuity** | Government Essential Functions | Single auth layer replacing fragmented legacy systems |

---

## 1.4 Guiding Principles / اصول راهنما

Every design decision is governed by the following principles in **strict priority order**:

1. **Privacy by Architecture** — Privacy is a mathematical guarantee, not a policy promise
2. **Sovereignty First** — No foreign entity has administrative access to Iranian citizen data
3. **Inclusion Without Exception** — Every Iranian has a path to enrollment regardless of geography, literacy, or documentation status
4. **Adversarial Security** — Designed assuming active subversion attempts by regime remnants and foreign actors
5. **Transparent System, Private Citizens** — Code and protocols are public; citizen data is not
6. **فارسی اول / Persian First** — Every interface designed RTL and Persian as the primary experience

---

# PART II: Stakeholder Analysis
# بخش دوم: تحلیل ذینفعان

## 2.1 Primary User Categories / دسته‌بندی کاربران اصلی

INDIS serves **five primary user categories**, each with distinct needs, technical capabilities, and trust relationships.

---

### 2.1.1 Category A: Individual Citizens / شهروندان

**Estimated Population:** 88 million residents + 8–10 million diaspora

#### Sub-Segments / زیرگروه‌ها

**Urban Digitally Literate Citizens / شهروندان شهری با سواد دیجیتال**

| Attribute | Detail |
|-----------|--------|
| Smartphone ownership | ~70% |
| Primary language | Persian |
| Secondary languages | Varies by ethnicity |
| Primary concerns | Privacy, ease of use, speed |
| Expected usage | Daily — banking, healthcare, government services |
| Technical literacy | Medium to High |

**Rural and Semi-Urban Citizens / شهروندان روستایی و نیمه‌شهری**

| Attribute | Detail |
|-----------|--------|
| Connectivity | Limited; feature phones common |
| Literacy | Variable; functional illiteracy present |
| Regional languages | Kurdish, Azerbaijani, Arabic, Baluchi, Gilaki, Mazandarani |
| Primary concerns | Physical access to enrollment, not being excluded |
| Special requirement | Offline enrollment capability is non-negotiable |

**Elderly Citizens / شهروندان سالمند**

| Attribute | Detail |
|-----------|--------|
| Digital literacy | Low across most of segment |
| Dependency | High on pension and healthcare systems |
| Documentation status | May have outdated or damaged documents |
| Design requirement | Simplified mode, assisted enrollment, audio guidance |

**Youth and Students / جوانان و دانشجویان**

| Attribute | Detail |
|-----------|--------|
| Digital literacy | Highest of all segments |
| Critical role | Referendum and election participation |
| Primary concern | Government surveillance; data privacy |
| Design requirement | Transparent privacy controls prominently displayed |

**Undocumented / Documentation-Deficient Citizens / شهروندان فاقد مدارک**

| Attribute | Detail |
|-----------|--------|
| Geographic concentration | Border regions, ethnic minority communities |
| Cause | Systematic exclusion from civil registration under Islamic Republic |
| Enrollment pathway | Social attestation (mandatory, not optional) |
| Political importance | Their inclusion is **non-negotiable** for transition legitimacy |

**Iranian Diaspora / ایرانیان خارج از کشور**

| Attribute | Detail |
|-----------|--------|
| Estimated size | 8–10 million in 50+ countries |
| Critical role | Referendum participation, professional reintegration |
| Enrollment pathway | Embassy and consulate network |
| Technical literacy | Generally high |
| Special requirement | Remote enrollment; multilingual support |

---

### 2.1.2 Category B: Government Entities / نهادهای دولتی

| Ministry / Body | Primary INDIS Use Case | Credential Types Accessed |
|-----------------|------------------------|---------------------------|
| Ministry of Interior | Civil registration, elections | Citizenship, Voter Eligibility |
| Ministry of Justice | Court records, criminal history | Full Identity (Level 3) |
| Ministry of Health | Coverage verification, prescriptions | Health Insurance |
| Ministry of Finance | Tax identity, pensions | Pension, Citizenship |
| Ministry of Education | Student enrollment, teacher credentialing | Professional, Age Range |
| Ministry of Foreign Affairs | Passport management | Citizenship, Diaspora |
| Transitional Mehestan | Member credentialing, voter rolls | Voter Eligibility |
| Transitional Divan | Proceeding authentication, evidence custody | Full Identity, Amnesty Applicant |
| NISS / Army / Police | Personnel vetting, access control | Security Clearance |
| Central Bank | Beneficial ownership, AML | Citizenship, Full Identity (Level 3) |

---

### 2.1.3 Category C: Verifiers / تأییدکنندگان

Verifiers confirm aspects of a citizen's identity **without necessarily seeing full identity data**. This is where ZK-proof technology delivers maximum value.

#### Verification Tiers / سطوح تأیید

| Level | Method | Use Case | Data Revealed to Verifier |
|-------|--------|----------|---------------------------|
| **Level 1** | QR scan + ZK proof | General service eligibility | Boolean result only |
| **Level 2** | NFC + biometric match | Financial transactions, voting | Credential category + validity status |
| **Level 3** | Full identity check | Border control, court proceedings | Full identity **with explicit citizen consent** |
| **Level 4** | Emergency override | Critical security situations | Full identity; requires senior auth + automatic audit alert |

**Private Sector Verifiers / تأییدکنندگان بخش خصوصی**
- Banks and financial institutions (KYC)
- Employers (work authorization)
- Insurance companies
- Telecommunications providers
- Real estate registries

**International Verifiers / تأییدکنندگان بین‌المللی**
- Foreign embassies (visa issuance)
- International organizations operating in Iran
- Foreign financial institutions (correspondent banking)

---

### 2.1.4 Category D: Enrollment Agents / عاملان ثبت‌نام

| Agent Type | Location | Connectivity | Authority Level |
|------------|----------|--------------|-----------------|
| Fixed Center Agents | Provincial offices, post offices, hospitals | High | Full enrollment |
| Mobile Unit Agents | Rural areas, border regions | Satellite/cellular | Full enrollment |
| Embassy Agents | 50+ countries | Variable | Full enrollment (diaspora) |
| Community Agents | Remote villages | Minimal | Initiate only; biometric required for completion |

---

### 2.1.5 Category E: Administrators and Auditors / مدیران و ممیزان

| Role | Access Level | Restrictions |
|------|-------------|--------------|
| NIA System Administrators | System configuration, key ceremony | Multi-party authorization for sensitive ops |
| Independent International Auditors | Read-only system logs | No access to citizen data |
| Parliamentary Oversight Committee | Aggregate statistics, audit reports | No individual citizen data |
| Security Red Team | Full penetration testing | Scheduled; isolated environment |

---

## 2.2 User Journey Maps / نقشه‌های سفر کاربر

### Journey 1: Urban Citizen — Self-Enrollment

```
┌─────────────────────────────────────────────────────────────┐
│  PRE-ENROLLMENT / پیش از ثبت‌نام                             │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
             Receives SMS/broadcast notification
             اطلاع‌رسانی از طریق پیامک یا رسانه
                          │
                          ▼
             Downloads INDIS app (iOS/Android/HarmonyOS)
             دانلود برنامه INDIS
                          │
                          ▼
             Selects language — Persian default (RTL)
             انتخاب زبان — پیش‌فرض فارسی
                          │
                          ▼
             Reads privacy notice — explicit consent captured
             مطالعه اطلاعیه حریم خصوصی — ثبت رضایت صریح
                          │
┌─────────────────────────▼───────────────────────────────────┐
│  DOCUMENT CAPTURE / ثبت مدارک                                │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
             Photographs existing documents (any combination):
             عکس‌برداری از مدارک موجود (هر ترکیبی قابل قبول):
             • Kart-e Melli (کارت ملی)
             • Shenasnameh (شناسنامه)
             • Passport / گذرنامه
             • Driver's License / گواهینامه
                          │
                          ▼
             AI-assisted document authenticity check
             بررسی صحت مدارک با هوش مصنوعی
                          │
┌─────────────────────────▼───────────────────────────────────┐
│  BIOMETRIC CAPTURE / ثبت بیومتریک                            │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
             Facial recognition + liveness detection
             تشخیص چهره + تشخیص زنده بودن
                          │
                          ▼
             10-finger fingerprint capture
             اثر انگشت ۱۰ انگشت
                          │
                          ▼
             Optional iris scan
             اسکن عنبیه (اختیاری)
                          │
┌─────────────────────────▼───────────────────────────────────┐
│  PROCESSING / پردازش                                         │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
             National biometric deduplication check
             بررسی تکراری نبودن بیومتریک در سطح ملی
             ⏱ Target: <90 seconds / هدف: کمتر از ۹۰ ثانیه
                          │
                          ▼
             DID generated ON-DEVICE
             تولید DID روی دستگاه کاربر
             🔑 Private key NEVER leaves device
             🔑 کلید خصوصی هرگز دستگاه را ترک نمی‌کند
                          │
┌─────────────────────────▼───────────────────────────────────┐
│  CREDENTIAL ISSUANCE / صدور مدارک دیجیتال                    │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
             Government issues Verifiable Credentials:
             صدور مدارک تأییدپذیر توسط دولت:
             ✓ Citizenship Credential / مدرک شهروندی
             ✓ Age Range Credential / مدرک محدوده سنی
             ✓ Residency Credential / مدرک محل سکونت
             ✓ Voter Eligibility Credential / مدرک واجد شرایط رأی
                          │
                          ▼
             Credentials stored in device wallet (encrypted)
             ذخیره مدارک در کیف پول دستگاه (رمزنگاری‌شده)
                          │
                          ▼
             Digital ID card generated + physical card option
             تولید کارت هویت دیجیتال + گزینه کارت فیزیکی
                          │
                          ▼
             ✅ ENROLLMENT COMPLETE / ثبت‌نام کامل شد
```

### Journey 2: Rural Undocumented Citizen — Social Attestation

```
┌─────────────────────────────────────────────────────────────┐
│  COMMUNITY AWARENESS / آگاهی‌سازی اجتماعی                    │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
             Mobile enrollment unit arrives in village
             واحد سیار ثبت‌نام به روستا می‌رسد
                          │
                          ▼
             Agent explains process in local language
             with visual aids (no literacy assumed)
             توضیح فرایند توسط عامل در زبان محلی
             با کمک‌های تصویری (بدون فرض سواد)
                          │
┌─────────────────────────▼───────────────────────────────────┐
│  SOCIAL ATTESTATION / تأیید اجتماعی                          │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
             3 enrolled community members co-attest identity
             ۳ عضو ثبت‌نام‌شده جامعه هویت را تأیید می‌کنند
             Each attestation cryptographically signed
             هر تأیید به‌صورت رمزنگاری امضا می‌شود
                          │
                          ▼
             Basic information captured:
             اطلاعات پایه ثبت می‌شود:
             • Name as known in community / نام شناخته‌شده
             • Approximate age / سن تقریبی
             • Village/region of origin / روستا/منطقه
             • Family relationships if known / روابط خانوادگی
                          │
                          ▼
             Biometric capture (same as standard)
             ثبت بیومتریک (مانند روش استاندارد)
                          │
                          ▼
             "Social Attestation" credential issued
             مدرک "تأیید اجتماعی" صادر می‌شود
             [Lower privilege; upgrade path displayed]
             [دسترسی محدود؛ مسیر ارتقا نمایش داده می‌شود]
                          │
                          ▼
             ✅ Basic services accessible immediately
             ✅ خدمات پایه بلافاصله در دسترس است
             📋 Voting eligibility pending secondary review
             📋 واجد شرایط رأی بودن در انتظار بررسی ثانویه
```

### Journey 3: ZK-Proof Verification at Service Counter

```
CITIZEN DEVICE           SERVICE COUNTER         BLOCKCHAIN
دستگاه شهروند            پیشخوان خدمات           بلاکچین
     │                        │                      │
     │   Verifier sends        │                      │
     │◄── verification ────────┤                      │
     │    request              │                      │
     │    [Credential type     │                      │
     │     + criteria only]    │                      │
     │                         │                      │
Citizen reviews              │                      │
approval screen              │                      │
شهروند صفحه                  │                      │
تأیید را می‌بیند             │                      │
     │                         │                      │
ZK circuit loads             │                      │
credential — generates       │                      │
proof ON-DEVICE              │                      │
تولید اثبات ZK               │                      │
روی دستگاه                   │                      │
⚡ No personal data          │                      │
   ever transmitted          │                      │
     │                         │                      │
     │   ZK Proof only         │                      │
     ├────────────────────────►│                      │
     │   (zero personal data)  │                      │
     │                         │  Proof verification  │
     │                         ├─────────────────────►│
     │                         │                      │
     │                         │  ✓ Valid / ✗ Invalid │
     │                         │◄─────────────────────┤
     │                         │                      │
     │                  ✅ or ❌ displayed             │
     │                  NO citizen data               │
     │                  seen by verifier              │
     │                  هیچ داده شخصی                │
     │                  نمایش داده نمی‌شود            │
```

---

# PART III: Functional Requirements
# بخش سوم: نیازمندی‌های کارکردی

## 3.1 Core Identity Management / مدیریت هویت مرکزی

### FR-001: Enrollment Processing / پردازش ثبت‌نام

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-001.1 | System SHALL process enrollment via three pathways: Standard (documents + biometrics), Enhanced (civil registry + biometrics), Social Attestation (community + biometrics) | MUST |
| FR-001.2 | Biometric deduplication SHALL complete within **90 seconds** under normal load | MUST |
| FR-001.3 | System SHALL generate a DID conforming to **W3C DID Core 1.0** for each enrolled individual | MUST |
| FR-001.4 | Private keys SHALL be generated **on citizen's device only**. Government servers SHALL NEVER hold citizen private keys | MUST |
| FR-001.5 | System SHALL support minor enrollment through parent/guardian with linked guardian credential | MUST |
| FR-001.6 | System SHALL issue a **temporary enrollment receipt** credential immediately after biometric capture | MUST |
| FR-001.7 | System SHALL support **bulk enrollment** for military, civil service, and hospital populations | SHOULD |
| FR-001.8 | Social attestation enrollment SHALL require minimum 3 co-attestors, each holding a valid INDIS credential of Tier 2 or above | MUST |
| FR-001.9 | Diaspora enrollment SHALL be available through the embassy network with full credential parity | MUST |

### FR-002: Credential Types / انواع مدارک دیجیتال

| Credential Type | نام فارسی | Issuing Authority | Validity | Key Attributes |
|-----------------|-----------|-------------------|----------|----------------|
| Citizenship | مدرک شهروندی | National Identity Authority | Permanent | Status, country-of-birth range, reg. date |
| Age Range | مدرک محدوده سنی | NIA | Permanent | Bracket only: <18, 18–30, 31–50, 51–70, >70 |
| Voter Eligibility | مدرک واجد شرایط رأی | Electoral Commission | Per election cycle | Eligibility boolean, district code |
| Residency | مدرک اقامت | Ministry of Interior | Annual | Province code, urban/rural |
| Professional | مدرک حرفه‌ای | Relevant Ministry | Defined by issuer | Category, qualification level |
| Health Insurance | مدرک بیمه سلامت | Ministry of Health | Annual | Coverage type, enrollment status |
| Pension | مدرک بازنشستگی | Ministry of Finance | Monthly | Beneficiary status, payment eligibility |
| Security Clearance | مدرک مجوز امنیتی | NIA / NISS | 90-day | Clearance level only (no details) |
| Amnesty Applicant | مدرک متقاضی عفو | Transitional Divan | Proceeding duration | Application status, case reference |
| Diaspora | مدرک ایرانی خارج از کشور | MFA | Annual | Diaspora status, country of residence |
| Social Attestation | مدرک تأیید اجتماعی | NIA + Community | 1 year (renewable) | Attestation level, upgrade path |

**FR-002 Rules:**
- **FR-002.R1:** Revocation propagation to all verifier nodes: ≤ 60 seconds
- **FR-002.R2:** Revocation registry SHALL be on-chain, checkable without querying central identity DB
- **FR-002.R3:** Selective disclosure MUST be supported for all credential types
- **FR-002.R4:** Expiry notifications: 30 days, 7 days, 1 day before expiry
- **FR-002.R5:** Credential delegation SHALL be supported with full audit trail

---

### FR-003: Zero-Knowledge Proof System / سیستم اثبات دانش صفر

**Architecture Decision:** ZKPs are the privacy backbone of INDIS. They are not optional features.

#### Proof Systems by Use Case

| Use Case | ZK System | Rationale |
|----------|-----------|-----------|
| Standard credential verification | **ZK-SNARKs (Groth16)** | Fast proof generation (<3s on mid-range phone) |
| Electoral / referendum verification | **ZK-STARKs** | Post-quantum security; no trusted setup required |
| Batch credential operations | **PLONK** | Universal trusted setup; efficient for bulk ops |
| Anonymous testimony (Justice) | **Bulletproofs** | No trusted setup; range proofs for age/date |

#### Standard ZK Proof Operations

**Age and Identity Proofs:**
```
prove_age_above(threshold: u8) → Proof
  // Proves age ≥ threshold without revealing exact age

prove_age_in_range(min: u8, max: u8) → Proof
  // Proves age within range without revealing exact age

prove_citizenship() → Proof
  // Proves Iranian citizenship without revealing any identifier

prove_voter_eligibility(election_id: Hash) → Proof
  // Atomic proof: citizenship + age ≥ 18 + not in exclusion list
  // Reveals: nothing beyond eligibility boolean
```

**Credential Validity Proofs:**
```
prove_credential_valid(credential_type: CredType) → Proof
  // Proves: issued by authorized issuer AND not revoked AND not expired

prove_credential_issued_within(start: Date, end: Date) → Proof
  // Proves issuance date within range without revealing exact date
```

**Selective Attribute Proofs:**
```
prove_attribute_meets_criteria(
  credential: Credential,
  attribute: AttributeKey,
  predicate: Predicate
) → Proof
  // Proves attribute satisfies predicate without revealing attribute value
```

**FR-003 Performance Targets:**

| Operation | Target | Maximum |
|-----------|--------|---------|
| ZK proof generation (standard) | 2 seconds | 5 seconds |
| ZK proof generation (electoral STARK) | 5 seconds | 15 seconds |
| Proof verification at terminal | 200ms | 500ms |
| Trusted setup ceremony participation | Multi-party; public | — |

**FR-003 Governance Requirements:**
- ALL ZK circuit code SHALL be **open source and publicly audited**
- Trusted setup (SNARK) SHALL be **multi-party computation** with international observers
- Formal verification of ZK circuits SHALL be completed before production deployment
- Circuit audit reports SHALL be published publicly

---

### FR-004: Biometric Management / مدیریت بیومتریک

| Modality | Required | Minimum Acceptable | Fallback |
|----------|----------|-------------------|----------|
| Facial recognition | Yes | 1 face | Audio-guided for visually impaired |
| Fingerprint | Yes | 4 fingers | If injury prevents full capture |
| Iris scan | Optional | N/A | Not required |
| Voice recognition | Optional | N/A | Accessibility pathway |

**Performance Requirements:**

| Metric | Target | Standard |
|--------|--------|----------|
| False Match Rate | ≤ 0.0001% | ISO/IEC 29794 |
| False Non-Match Rate | ≤ 0.1% | ISO/IEC 29794 |
| Liveness Detection (IAPAR) | ≤ 0.5% | ISO/IEC 30107-3 |
| Presentation Attack Detection | Mandatory | ISO/IEC 30107-3 |

**FR-004 Privacy Rules:**
- Biometric templates stored ONLY in encrypted form (AES-256-GCM, HSM-managed keys)
- One-way transformation applied before storage — original cannot be reconstructed
- Biometric data SHALL NEVER be shared with any foreign government, organization, or private entity
- Alternative pathways mandatory for persons with disabilities or occupational biometric wear

---

## 3.2 Citizen Application / برنامه شهروندی

### FR-005: Platform Support / پشتیبانی از سکوها

| Platform | Minimum Version | Notes |
|----------|----------------|-------|
| Android | 8.0 (API 26) | Primary; highest Iranian market share |
| iOS | 14.0 | Secondary |
| HarmonyOS | 2.0 | Required for Huawei devices common in Iran |
| Progressive Web App | Modern browsers | Fallback for unsupported devices |
| USSD / SMS | Feature phones | Essential for rural low-connectivity access |

### FR-006: Language and RTL Support / پشتیبانی از زبان و راست‌به‌چپ

**CRITICAL: Persian is designed first. All other language interfaces are derived from the Persian/RTL design, not the reverse.**

| Language | Script | Direction | Priority | Status |
|----------|--------|-----------|----------|--------|
| **فارسی (Persian)** | **Naskh / Nastaliq** | **RTL** | **Primary** | **Launch** |
| **English** | Latin | LTR | **Co-Primary** | **Launch** |
| کردی سورانی (Kurdish Sorani) | Modified Arabic | RTL | Tier 1 | Launch |
| کردی کرمانجی (Kurdish Kurmanji) | Latin | LTR | Tier 1 | Launch |
| آذربایجانی (Azerbaijani Turkish) | Latin / Arabic | LTR / RTL | Tier 1 | Launch |
| عربی (Arabic — Iranian dialect) | Arabic | RTL | Tier 1 | Launch |
| بلوچی (Baluchi) | Arabic | RTL | Tier 1 | Phase 2 |
| فرانسه (French) | Latin | LTR | Tier 2 | Phase 3 |

**Localization Technical Requirements:**

| Requirement | Specification |
|-------------|---------------|
| Calendar system | Solar Hijri (Shamsi) default; Gregorian secondary |
| Number display | Persian numerals (۰۱۲۳۴۵۶۷۸۹) default; switchable |
| Date format | روز/ماه/سال with full Persian month names |
| Typography | **Vazirmatn** (open-source, comprehensive Unicode, excellent mobile legibility) |
| RTL cursor | Correct right-to-left cursor movement |
| Bidirectional text | Correct mixed Persian-English rendering (Unicode Bidi algorithm) |
| Search/sort | Persian alphabetical order (الفبا), not Unicode code point order |
| Name handling | Supports Arabic-origin names, compound names, names without family names |

### FR-007: Digital Identity Card Display / نمایش کارت هویت دیجیتال

The primary screen SHALL display a digital identity card styled in **Iranian cultural heritage visual language**.

**Card Display Elements:**

```
┌────────────────────────────────────────────────┐
│  سیستم هویت دیجیتال ملی ایران               │
│  Iran National Digital Identity System          │
├────────────────────────────────────────────────┤
│                                                  │
│   [PHOTO]    نام خانوادگی، نام               │
│              Family Name, Given Name             │
│                                                  │
│              کد ملی: ●●●●●●●●●● [نمایش]       │
│              National ID: ●●●●●●●●●● [Reveal]  │
│                                                  │
│   ✓ شهروندی  ✓ رأی‌دهنده  ✓ بیمه سلامت       │
│   ✓ Citizen  ✓ Voter     ✓ Health Ins.         │
│                                                  │
│   [QR — tap biometric to reveal]                │
│   [کیوآر — لمس بیومتریک برای نمایش]            │
│                                                  │
│   وضعیت: تأییدشده ✅  |  Status: Verified ✅  │
└────────────────────────────────────────────────┘
```

### FR-008: Privacy Control Center / مرکز کنترل حریم خصوصی

The Privacy Control Center SHALL be a **prominently accessible** feature — not buried in settings. This is a core political commitment.

**Required Displays:**
- Complete history of all verification requests received
- Complete history of all credentials shared
- List of verifier categories that accessed credentials in past 12 months
- Data minimization settings per service category
- Per-verifier sharing rules: Always Share / Always Ask (default) / Never Share

**Citizen Rights Enforced by UI:**
- Real-time alert for every credential verification attempt
- One-tap decline for any verification request
- Complete data export (cryptographically signed package, 72-hour delivery)
- Data correction request with tracked resolution
- Escalation pathway to an independent ombudsman

---

## 3.3 Government Portal / درگاه دولتی

### FR-009: Ministry Integration Dashboard

| Requirement | Detail |
|-------------|--------|
| Authentication | Certificate-based (no password-only access) |
| Data minimization | Role-based data visibility; pension officer CANNOT see security clearance |
| Audit logging | Every data access logged; retained 10 years minimum |
| Bulk operations | Require senior approval for operations affecting >1,000 records |
| Approval chains | Multi-level, configurable per credential type |

### FR-010: Electoral Authority Module / ماژول هیأت انتخابات

**This module has a hard deadline: operational before the constitutional referendum (≤4 months post-transition)**

| Property | Implementation |
|----------|----------------|
| **Completeness** | Every valid vote counted — cryptographic proof |
| **Soundness** | No invalid vote counted — ZK-STARK verification |
| **Privacy** | No one can determine how any individual voted |
| **Individual Verifiability** | Every voter can verify their vote was counted |
| **Universal Verifiability** | Any mathematician can verify total result |
| **Receipt-Freeness** | Cannot prove to third party how you voted (anti-coercion) |

Both **in-person** (QR scan at polling station) and **remote voting** (cryptographic ballot) SHALL be supported.

### FR-011: Transitional Justice Module / ماژول عدالت انتقالی

**Anonymous Testimony System:**
- Witness proves Iranian citizenship (ZK proof) without revealing identity
- Cryptographic receipt issued on submission
- Follow-up testimony linkable to original without identity disclosure
- Optional identity reveal to sealed court record only, with explicit consent

**Conditional Amnesty Workflow:**
- Verified full identity required (applicant cannot use ZK anonymity)
- Victim notification system (notifies victims of application without revealing applicant identity prematurely)
- Multi-party review with mandatory recusal checking
- Additional encryption layer; keys in judicial multi-party escrow

---

## 3.4 Verifier Application / برنامه تأییدکننده

### FR-012: Verifier Registration and Authorization

Every verifier organization SHALL register with the NIA and receive a **verifier certificate** defining:
- Which credential types it is authorized to request
- Which attributes within credentials it may verify
- Maximum verification frequency limits
- Permitted verification contexts
- Geographic scope (nationwide / provincial / site-specific)

### FR-013: Verification Result Display

**CRITICAL: Verifier terminals SHALL display ONLY a binary result for ZK-proof verifications.**

```
APPROVED                           DENIED
تأیید شد                          رد شد

[GREEN SCREEN]                    [RED SCREEN]
✅                                 ❌

Voter eligibility: CONFIRMED      Voter eligibility: NOT CONFIRMED
واجد شرایط رأی: تأیید شد         واجد شرایط رأی: تأیید نشد

No citizen data displayed.        No reason displayed to verifier.
هیچ داده شخصی نمایش داده نمی‌شود  دلیل به تأییدکننده نمایش داده نمی‌شود
```

---

# PART IV: Non-Functional Requirements
# بخش چهارم: نیازمندی‌های غیرکارکردی

## 4.1 Performance Requirements / نیازمندی‌های عملکردی

| Operation | Target | Maximum | Notes |
|-----------|--------|---------|-------|
| Credential verification (online) | 500ms | 2s | — |
| Credential verification (cached offline) | 100ms | 500ms | Up to 72h offline |
| ZK proof generation (standard Groth16) | 2s | 5s | On 2020 mid-range Android |
| ZK proof generation (electoral STARK) | 5s | 15s | — |
| Proof verification at terminal | 200ms | 500ms | — |
| Biometric deduplication | 30s | 90s | Full national population |
| Blockchain write finality | 1s | 3s | — |
| App launch to ready state | 2s | 5s | Cold start |
| Credential issuance | 5s | 30s | — |
| Bulk enrollment processing | 500K/day | — | Peak campaign |
| Electoral event verification | 2M/hour | — | Referendum day |

## 4.2 Availability Requirements / نیازمندی‌های دسترس‌پذیری

| Service Tier | Uptime SLA | Max Downtime/Year | RPO | RTO |
|-------------|-----------|-------------------|-----|-----|
| Core identity verification | 99.99% | 53 minutes | 15 min | 1 hour |
| Enrollment services | 99.9% | 8.7 hours | 1 hour | 4 hours |
| Electoral services (active election) | 99.999% | 5 minutes | 5 min | 30 min |
| Non-critical reporting | 99.5% | 44 hours | 4 hours | 8 hours |

## 4.3 Security Requirements / نیازمندی‌های امنیتی

### Cryptographic Standards

| Component | Standard | Notes |
|-----------|----------|-------|
| Data in transit | TLS 1.3 minimum | TLS 1.2 only as legacy fallback |
| Data at rest | AES-256-GCM | — |
| Digital signatures | Ed25519 or ECDSA P-256 | — |
| Key management | FIPS 140-2 Level 3 HSM | Or equivalent |
| Long-term credentials | NIST PQC (CRYSTALS-Dilithium) | Post-quantum readiness |
| Cryptographic libraries | Audited open-source ONLY | No proprietary crypto |
| ZK circuits | Formally verified | Before production |

### Threat Model

| Threat | Attack Vector | Mitigation |
|--------|--------------|------------|
| **Regime Remnant Manipulation** | Corrupt agents creating fraudulent identities for persons evading accountability | Agent credential signing; pattern anomaly detection; mandatory supervisory review for social attestation |
| **Foreign State Cyberattack** | Nation-state attack on electoral infrastructure | Air-gapped backups; distributed architecture; international security partnerships; regular red-teaming |
| **Identity Fraud at Scale** | Multiple identity enrollment for vote manipulation or benefit fraud | Biometric deduplication at enrollment; liveness detection; cross-database reconciliation |
| **Future Government Overreach** | Next authoritarian government using INDIS for mass surveillance | ZK-proof architecture makes surveillance technically impossible at protocol level; citizen-controlled private keys; parliamentary oversight with technical enforcement |
| **Insider Threat** | NIA administrators abusing privileged access | RBAC with minimum privilege; all access logged; multi-party authorization for sensitive ops; separation of duties |

### Audit and Testing Requirements

| Requirement | Specification |
|-------------|---------------|
| Pre-launch security audits | Minimum 2 independent internationally recognized firms |
| Automated security scanning | Continuous; results published monthly |
| Bug bounty program | Public; established before production launch |
| Formal ZK circuit verification | Required before deployment |
| Red team exercises | Quarterly; scenarios include regime-remnant and nation-state |
| Penetration testing | Annual; scope covers all user-facing surfaces |

## 4.4 Privacy Requirements / نیازمندی‌های حریم خصوصی

| Requirement | Detail |
|-------------|--------|
| **Data Minimization** | Collect only strictly necessary data per function |
| **Purpose Limitation** | Enrollment data used ONLY for identity verification — not law enforcement without judicial order |
| **Cross-Verifier Correlation** | Architecture SHALL make cross-verifier behavioral profiling technically impossible without judicial authorization |
| **Differential Privacy** | Aggregate population statistics use differential privacy (ε parameter published publicly) |
| **Biometric Sovereignty** | Biometric data NEVER shared with any foreign entity under any circumstances |
| **DPIA** | Published before each major component goes to production |

## 4.5 Offline and Low-Connectivity Requirements

| Capability | Detail |
|------------|--------|
| Citizen app offline | Full credential presentation and ZK proof generation without network |
| Verifier offline | Up to 72 hours using cached revocation lists |
| Enrollment agent offline | Complete capture with queued sync |
| SMS fallback | USSD / SMS short codes for basic eligibility checks on feature phones |
| Physical card | ISO 7816 embedded chip; offline verification via card-reader terminals |

---

# PART V: System Architecture
# بخش پنجم: معماری سیستم

## 5.1 High-Level Architecture

```
╔═══════════════════════════════════════════════════════════════════╗
║                     CITIZEN LAYER / لایه شهروند                   ║
║  ┌────────────┐  ┌────────────┐  ┌──────────┐  ┌──────────────┐  ║
║  │ Mobile App │  │  Web PWA   │  │  Kiosk   │  │ Physical Card│  ║
║  │iOS/Android │  │  Persian   │  │ (Post    │  │ ISO 7816/NFC │  ║
║  │HarmonyOS   │  │  RTL-first │  │ Offices) │  │              │  ║
║  └─────┬──────┘  └─────┬──────┘  └────┬─────┘  └──────┬───────┘  ║
╚════════╪════════════════╪══════════════╪════════════════╪══════════╝
         │                │              │                │
╔════════╪════════════════╪══════════════╪════════════════╪══════════╗
║        └────────────────┴──────────────┴────────────────┘          ║
║                     API GATEWAY / دروازه API                       ║
║  ┌────────────────────────────────────────────────────────────┐    ║
║  │ Rate Limiting │ mTLS Auth │ Routing │ Load Balancing │ WAF │    ║
║  └────────────────────────────────────────────────────────────┘    ║
╚══════════════════════════════════╤════════════════════════════════╝
                                   │
╔══════════════════════════════════╪════════════════════════════════╗
║              CORE SERVICES LAYER / لایه خدمات مرکزی               ║
║                                  │                                 ║
║  ┌───────────────┐  ┌────────────┴──┐  ┌───────────────────────┐  ║
║  │   Identity    │  │  Credential   │  │    ZK Proof Service   │  ║
║  │   Service     │  │   Service     │  │  (Groth16 / STARK /   │  ║
║  │               │  │               │  │   PLONK / Bulletproof)│  ║
║  └───────────────┘  └───────────────┘  └───────────────────────┘  ║
║                                                                     ║
║  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────┐  ║
║  │   Biometric   │  │  Enrollment   │  │   Notification        │  ║
║  │   Service     │  │   Service     │  │   Service             │  ║
║  │               │  │               │  │ (SMS/Push/Email)      │  ║
║  └───────────────┘  └───────────────┘  └───────────────────────┘  ║
║                                                                     ║
║  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────┐  ║
║  │   Audit       │  │   Electoral   │  │   Justice Module      │  ║
║  │   Service     │  │   Module      │  │   (ZK Testimony +     │  ║
║  │               │  │   (STARK-ZK)  │  │    Amnesty Workflow)  │  ║
║  └───────────────┘  └───────────────┘  └───────────────────────┘  ║
╚══════════════════════════════════╤════════════════════════════════╝
                                   │
╔══════════════════════════════════╪════════════════════════════════╗
║                DATA LAYER / لایه داده                              ║
║                                  │                                 ║
║  ┌────────────────┐  ┌───────────┴───────────┐  ┌──────────────┐  ║
║  │  Identity DB   │  │  BLOCKCHAIN LAYER     │  │  Biometric   │  ║
║  │  PostgreSQL +  │  │  [TBD — Hyperledger   │  │  DB          │  ║
║  │  TimescaleDB   │  │   Fabric candidate]   │  │  Air-gapped  │  ║
║  │  (Encrypted)   │  │                       │  │  HSM keys    │  ║
║  └────────────────┘  │  • DID Registry       │  └──────────────┘  ║
║                       │  • Credential Anchors │                    ║
║  ┌────────────────┐  │  • Revocation Status  │  ┌──────────────┐  ║
║  │  Audit Log     │  │  • No personal data   │  │  Key Mgmt    │  ║
║  │  Append-only   │  └───────────────────────┘  │  HSM Cluster │  ║
║  │  Crypto-signed │                              │  FIPS 140-2  │  ║
║  └────────────────┘                              └──────────────┘  ║
╚═════════════════════════════════════════════════════════════════════╝
```

## 5.2 ZK Proof Data Flow

```
┌──────────────────────────────────────────────────────────────────┐
│              ZK PROOF GENERATION AND VERIFICATION FLOW           │
│         جریان تولید و تأیید اثبات دانش صفر                       │
└──────────────────────────────────────────────────────────────────┘

1. CREDENTIAL STORAGE (at enrollment / هنگام ثبت‌نام)
   ─────────────────────────────────────────────────
   Government issues VC → NIA signs with private key
                       → Credential anchored on blockchain (hash only)
                       → Credential delivered to citizen device
                       → Stored in encrypted device wallet
                       → NIA private key NEVER on citizen device
                       → Citizen private key NEVER on government server

2. VERIFICATION REQUEST (at service point / در نقطه خدمت)
   ──────────────────────────────────────────────────────
   Verifier terminal → Sends: CredentialType + Predicate + Nonce
                       e.g., "Prove voter eligibility for election #7"
                       NO citizen data sent in request

3. ZK PROOF GENERATION (on citizen device / روی دستگاه شهروند)
   ──────────────────────────────────────────────────────────────
   Device loads ZK circuit for requested proof type
   Circuit inputs: [credential, predicate, nonce]
   Circuit outputs: [proof] — contains ZERO personal data
   Generation time: <5 seconds
   Citizen approves sharing → Proof sent to verifier

4. VERIFICATION (at verifier + blockchain / تأیید)
   ─────────────────────────────────────────────────
   Verifier receives proof
   Checks proof against:
     (a) ZK verification key (public, downloaded from NIA)
     (b) Revocation status (queried from blockchain)
     (c) Credential anchor (queried from blockchain)
   Result: ✅ Valid or ❌ Invalid
   NO citizen data processed by verifier at any step

5. AUDIT (citizen-controlled / کنترل شهروند)
   ────────────────────────────────────────────
   Verification event logged in citizen's own wallet
   Citizen sees: date, verifier category, credential used
   Verifier sees: boolean result only
   NIA sees: anonymized aggregate statistics only
```

## 5.3 Blockchain Abstraction Layer

> ⚠️ **The blockchain provider has not been selected.** This section defines the abstraction layer that decouples application logic from the specific blockchain implementation, enabling Hyperledger Fabric adoption or substitution with minimal application-layer changes.

### Abstraction Interface

```typescript
// Blockchain Abstraction Interface
// All blockchain interactions go through this interface ONLY
// No application service SHALL call blockchain SDK directly

interface BlockchainAdapter {
  
  // DID Operations
  registerDID(did: string, document: DIDDocument): Promise<TxReceipt>
  resolveDID(did: string): Promise<DIDDocument>
  updateDIDDocument(did: string, update: Partial<DIDDocument>): Promise<TxReceipt>
  deactivateDID(did: string): Promise<TxReceipt>
  
  // Credential Anchoring
  anchorCredential(credentialHash: Hash, issuerDID: string): Promise<TxReceipt>
  verifyAnchor(credentialHash: Hash): Promise<AnchorStatus>
  
  // Revocation Registry
  revokeCredential(credentialId: string, reason: RevocationReason): Promise<TxReceipt>
  checkRevocationStatus(credentialId: string): Promise<RevocationStatus>
  getRevocationList(issuerDID: string): Promise<RevocationList>
  
  // Audit Trail (anonymized)
  logVerificationEvent(event: AnonymizedVerificationEvent): Promise<TxReceipt>
  
  // Health and Status
  getBlockHeight(): Promise<number>
  getValidatorStatus(): Promise<ValidatorStatus[]>
  estimateTxTime(): Promise<milliseconds>
}
```

### Hyperledger Fabric Implementation Notes

When Hyperledger Fabric is selected, the following design decisions apply:

| Decision | Specification |
|----------|---------------|
| Network topology | Private permissioned; NIA as ordering service admin |
| Consensus | Raft (CFT) for performance; PBFT if Byzantine fault tolerance required |
| Channel structure | Separate channels per credential type for data isolation |
| Chaincode language | **Go** (primary) + TypeScript (secondary) |
| Identity MSP | Fabric CA integrated with NIA PKI |
| State database | **CouchDB** for rich queries on credential status |
| Block time | Target: 500ms; Maximum: 2 seconds |
| Endorsement policy | Minimum 3-of-5 NIA nodes for credential operations |
| Peer node distribution | Minimum 21 nodes; geographically distributed |
| Privacy extensions | Fabric Private Data Collections for sensitive credential metadata |
| No personal data on-chain | ENFORCED by chaincode — hash only; plaintext rejected |

### Blockchain Selection Evaluation Criteria

The final selection should evaluate candidate platforms against:

| Criterion | Weight | Notes |
|-----------|--------|-------|
| Iranian sovereignty (no foreign control) | 25% | Critical — must eliminate vendor admin access |
| Throughput (TPS at production load) | 20% | Minimum 10,000 TPS for election day |
| Finality time | 15% | Target <3 seconds |
| HSM integration | 15% | FIPS 140-2 Level 3 required |
| Open source maturity | 10% | Active community; auditable |
| Operational complexity | 10% | Must be operable by Iranian technical teams |
| International standards compliance | 5% | W3C DID, VC Data Model |

**Current Candidate Ranking:**

| Platform | Sovereignty | TPS | Finality | Recommendation |
|----------|-------------|-----|----------|----------------|
| **Hyperledger Fabric** | ✅ Full control | ~3,500 base; higher with tuning | <2s | **Primary candidate** |
| Hyperledger Besu | ✅ Full control | ~1,000-2,000 | 2-5s | Secondary candidate |
| Custom BFT (purpose-built) | ✅ Full control | Highest | <1s | Long-term option post-stabilisation |
| Public Ethereum / L2 | ❌ No sovereignty | Variable | Variable | Not suitable |

---

# PART VI: Technology Stack
# بخش ششم: پشته فناوری

## 6.1 Stack Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    INDIS TECHNOLOGY STACK                        │
│               پشته فناوری سیستم هویت ملی ایران                  │
├──────────────────┬──────────────────────────────────────────────┤
│  LAYER / لایه    │  TECHNOLOGY CHOICES                          │
├──────────────────┼──────────────────────────────────────────────┤
│ Mobile (iOS)     │ Swift / SwiftUI                              │
│ Mobile (Android) │ Kotlin / Jetpack Compose                     │
│ Mobile (Harmony) │ ArkTS / ArkUI                                │
│ Mobile (PWA)     │ React + TypeScript (RTL-first)               │
├──────────────────┼──────────────────────────────────────────────┤
│ ZK Circuits      │ Circom 2.0 + SnarkJS (Groth16/PLONK)        │
│                  │ StarkWare Cairo (STARKs for elections)       │
│                  │ Bulletproofs-rs (anonymous testimony)        │
├──────────────────┼──────────────────────────────────────────────┤
│ Backend Services │ Go (primary — performance-critical services) │
│                  │ Rust (ZK proof service, crypto operations)   │
│                  │ Python (ML/AI services, data analytics)      │
├──────────────────┼──────────────────────────────────────────────┤
│ API Layer        │ gRPC (internal service communication)        │
│                  │ REST + OpenAPI 3.0 (external integrations)   │
│                  │ GraphQL (government portal queries)          │
├──────────────────┼──────────────────────────────────────────────┤
│ API Gateway      │ Kong (open-source) or NGINX + custom        │
│                  │ mTLS for all service-to-service calls        │
├──────────────────┼──────────────────────────────────────────────┤
│ Identity DB      │ PostgreSQL 15+ (primary relational store)   │
│                  │ TimescaleDB extension (audit time-series)    │
│                  │ Redis (session cache, revocation cache)      │
├──────────────────┼──────────────────────────────────────────────┤
│ Biometric DB     │ PostgreSQL + pgvector (template storage)    │
│                  │ Air-gapped deployment; HSM key management    │
│                  │ Neurotechnology SDK or open NIST-compliant  │
├──────────────────┼──────────────────────────────────────────────┤
│ Blockchain       │ [TBD] — Hyperledger Fabric primary candidate│
│                  │ Abstraction layer enforced (see §5.3)        │
│                  │ CouchDB state DB (Fabric)                   │
├──────────────────┼──────────────────────────────────────────────┤
│ Key Management   │ HashiCorp Vault + HSM backend               │
│                  │ AWS CloudHSM / Thales Luna (if partnership) │
│                  │ Custom national HSM if sovereignty requires  │
├──────────────────┼──────────────────────────────────────────────┤
│ Infrastructure   │ Kubernetes (container orchestration)        │
│                  │ Helm (deployment management)                 │
│                  │ Terraform (infrastructure as code)          │
│                  │ Istio service mesh (mTLS, observability)    │
├──────────────────┼──────────────────────────────────────────────┤
│ Observability    │ Prometheus + Grafana (metrics)              │
│                  │ OpenTelemetry (distributed tracing)         │
│                  │ ELK Stack (log aggregation — privacy-safe)  │
├──────────────────┼──────────────────────────────────────────────┤
│ CI/CD            │ GitLab CI (self-hosted)                      │
│                  │ ArgoCD (GitOps deployment)                   │
│                  │ SonarQube (code quality)                     │
│                  │ Trivy (container security scanning)          │
├──────────────────┼──────────────────────────────────────────────┤
│ AI / ML          │ Python + PyTorch (biometric deduplication)  │
│                  │ ONNX Runtime (on-device inference)          │
│                  │ scikit-learn (fraud pattern detection)       │
│                  │ Apache Kafka (real-time event streaming)     │
├──────────────────┼──────────────────────────────────────────────┤
│ Biometric SDK    │ NIST BOZORTH3 (fingerprint matching)        │
│                  │ OpenFace / ArcFace (facial recognition)     │
│                  │ ISO 30107-3 compliant liveness detection    │
├──────────────────┼──────────────────────────────────────────────┤
│ Standards        │ W3C DID Core 1.0                            │
│                  │ W3C Verifiable Credentials 2.0              │
│                  │ OpenID Connect 4 Verifiable Presentations   │
│                  │ ISO/IEC 18013-5 (mDL — mobile ID)          │
│                  │ ICAO 9303 (physical card)                   │
│                  │ ISO/IEC 30107-3 (PAD / liveness)           │
│                  │ FIPS 140-2 Level 3 (HSM)                   │
│                  │ NIST PQC (post-quantum)                     │
└──────────────────┴──────────────────────────────────────────────┘
```

## 6.2 Technology Rationale / دلایل انتخاب فناوری

### Why Go for Backend Services?
Go provides the right balance of performance, concurrency (critical for biometric deduplication under load), and operational simplicity. The standard library is comprehensive, deployment is a single binary, and the language is well-suited for building high-reliability government infrastructure. The Iranian developer community has growing Go expertise.

### Why Rust for ZK and Crypto?
Cryptographic operations and ZK proof generation are memory-safety-critical. Rust's ownership model eliminates entire classes of vulnerabilities (buffer overflows, use-after-free) that are catastrophic in cryptographic code. The Rust ZK ecosystem (arkworks, bellman, bulletproofs-rs) is mature and actively audited.

### Why Circom + SnarkJS for ZK Circuits?
Circom is purpose-built for ZK circuit development with strong tooling and a large audited circuit library. SnarkJS provides browser and Node.js proving, enabling both server-side and on-device proof generation from a single circuit codebase.

### Why Vazirmatn for Persian Typography?
- Open-source (OFL license) — no licensing costs or restrictions
- Complete Unicode coverage including all Persian, Arabic, and extended characters
- Specifically optimized for screen rendering at small sizes on mobile devices
- Maintained by the Iranian open-source community
- Available in multiple weights suitable for both display and body text

### Why Self-Hosted GitLab CI?
All source code, CI/CD pipelines, and deployment tooling must be under Iranian sovereign control. Cloud-hosted services (GitHub Actions, CircleCI) create foreign dependencies incompatible with the sovereignty requirement.

## 6.3 What We Are NOT Using

| Technology | Reason Excluded |
|------------|-----------------|
| AWS / Azure / Google Cloud | Foreign government subpoena risk; sovereignty violation |
| Any closed-source cryptographic library | Cannot be audited; trust cannot be established |
| Public blockchains (Ethereum, Solana, etc.) | No sovereignty; unpredictable costs; foreign validator control |
| Biometric SDKs without source audit rights | Cannot verify absence of backdoors |
| Any technology with Iranian sanctions exposure | Legal risk to international technical partners |

---

# PART VII: Implementation Roadmap
# بخش هفتم: نقشه راه پیاده‌سازی

## 7.1 Phase Overview

```
MONTH:  1    2    3    4    5    6    7    8    9   10   11   12   18   24
        │    │    │    │    │    │    │    │    │    │    │    │    │    │
        ├────┤                                                           │
PHASE 0 │████│ Foundation and governance                                 │
        ├────┴──┤                                                        │
PHASE 1        │███████│ Priority enrollment                             │
               ├───────┤                                                 │
PHASE 2        │  Hard deadline: referendum ready ◄──────────────────── │
               │        │                                                │
               ├────────┴──────────────────┤                            │
PHASE 3                                    │████████████│ National rollout
                                           ├────────────┴───────────────┤
PHASE 4                                                                  │███
                                                              Full coverage│
```

## 7.2 Phase 0 — Foundation (Months 1–2)

**Governance Establishment:**
- [ ] Establish National Identity Authority (NIA) as governing body
- [ ] Conduct ZK trusted setup ceremony with international observers (SNARK)
- [ ] Open-source publication of all core cryptographic code
- [ ] Publish Data Protection Impact Assessment (DPIA)
- [ ] Establish Parliamentary Oversight Committee with technical staff
- [ ] Publish public bug bounty program
- [ ] Initiate blockchain platform evaluation and selection process

**Technical Foundation:**
- [ ] Deploy blockchain validator nodes (minimum 21, geographically distributed)
- [ ] Establish HSM cluster for key management (FIPS 140-2 Level 3)
- [ ] Deploy core identity and credential services in staging environment
- [ ] Complete initial penetration testing of core infrastructure
- [ ] Deploy CI/CD pipeline (self-hosted GitLab)
- [ ] Establish DR sites and test failover procedures

**Deliverables:**
- Functioning staging environment
- Published security audit results
- Governance framework operational
- Blockchain platform selected OR evaluation criteria published with timeline

---

## 7.3 Phase 1 — Priority Enrollment (Months 2–3)

**Target Population:** Military/security personnel, civil servants, healthcare workers  
**Hard Requirement:** Military vetting operational by Day 40 (Emergency Phase Booklet — Military Chapter)  
**Scale Target:** 2 million enrollments

**Technical Milestones:**
- [ ] Mobile enrollment application: iOS + Android in Persian (RTL-first) and English
- [ ] Enrollment agent application: 50+ fixed enrollment centers
- [ ] Biometric deduplication operational for up to 2 million population
- [ ] Initial credential types live: Citizenship, Age Range, Security Clearance
- [ ] Government portal operational for Ministry of Interior and Defense
- [ ] Bulk enrollment pipeline operational

**Definition of Done:**
- 100% of defined priority personnel enrolled and credentialed
- Security clearance credential issued and verifiable
- Zero critical security findings in production environment

---

## 7.4 Phase 2 — Electoral Preparation (Months 3–4)

> ⚠️ **HARD DEADLINE:** The constitutional referendum is required within 4 months of transition. The electoral module must be operational and independently audited before this date. This deadline is non-negotiable.

**Technical Milestones:**
- [ ] Electoral module: fully operational and independently audited (STARK-ZK)
- [ ] Voter eligibility credential: issued to all Phase 1 enrolled citizens
- [ ] Remote voting capability: deployed and penetration tested
- [ ] International observer access tools: deployed
- [ ] SMS/USSD verification fallback: operational for low-connectivity regions
- [ ] Diaspora voting portal: operational through embassy network

**Electoral System Audit Requirements:**
- Independent audit by minimum 2 internationally recognized firms
- MUST complete minimum 14 days before referendum date
- Audit reports published publicly
- International observer participation in verification process
- End-to-end verifiability demonstrated publicly before election day

**Scale Target:** 10 million enrollments (full urban coverage)

---

## 7.5 Phase 3 — National Rollout (Months 4–12)

**Technical Milestones:**
- [ ] Mobile enrollment units: deployed to all 31 provinces
- [ ] All credential types: operational
- [ ] Pension and subsidy payment integration: complete
- [ ] Healthcare and pharmacy verification: deployed
- [ ] Full minority language support: Kurdish, Azerbaijani, Arabic, Baluchi
- [ ] Transitional Justice module: operational (Truth Commission + Amnesty)
- [ ] Private sector verifier program: launched (banks, telecoms)
- [ ] Physical card distribution: nationwide

**Scale Target:** 50 million enrollments

---

## 7.6 Phase 4 — Full Coverage (Months 12–24)

**Technical Milestones:**
- [ ] Diaspora enrollment: complete through full embassy network
- [ ] Physical card distribution: complete for non-smartphone populations
- [ ] International interoperability: framework with partner countries
- [ ] Post-quantum migration: all long-term credentials migrated to CRYSTALS-Dilithium
- [ ] System optimization: based on 12 months of production telemetry
- [ ] Full open-source publication: all components

**Scale Target:** 85+ million enrollments (full coverage)

---

# PART VIII: Blockchain Framework Detail
# بخش هشتم: جزئیات چارچوب بلاکچین

## 8.1 What the Blockchain Stores (and What It Does NOT)

```
STORES ✅                          DOES NOT STORE ❌
──────────────────────────         ─────────────────────────────
DID Document (public key,          Names
  service endpoints)               Addresses
Credential hash anchors            National ID numbers
Revocation status (boolean)        Biometric data
Anonymized verification events     Any credential content
Validator node status              Photos
System health metrics              Any personal information
                                   Any linkable identifier
```

**This separation is enforced at the chaincode level. Chaincode will reject any transaction containing data that matches personal information patterns.**

## 8.2 Hyperledger Fabric Network Topology

```
┌─────────────────────────────────────────────────────────────────┐
│              HYPERLEDGER FABRIC NETWORK TOPOLOGY                 │
│          (IF FABRIC IS SELECTED AS BLOCKCHAIN PLATFORM)         │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ORDERING SERVICE (Raft)                                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │ Orderer 1│ │ Orderer 2│ │ Orderer 3│ │ Orderer 4│           │
│  │ Tehran   │ │ Isfahan  │ │ Mashhad  │ │ Tabriz   │           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
│                                                                   │
│  PEER NODES (minimum 21, geographically distributed)            │
│  ┌─────────────────────────────────────────────────────┐        │
│  │ Org: NIA    │ Org: MoI    │ Org: MoH   │ Org: MoF  │        │
│  │ 6 peers     │ 5 peers     │ 5 peers    │ 5 peers   │        │
│  └─────────────────────────────────────────────────────┘        │
│                                                                   │
│  CHANNELS                                                        │
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐    │
│  │ did-registry    │  │ credential-     │  │ electoral    │    │
│  │ channel         │  │ anchor channel  │  │ channel      │    │
│  │ (all orgs)      │  │ (all orgs)      │  │ (NIA + MoI)  │    │
│  └─────────────────┘  └─────────────────┘  └──────────────┘    │
│                                                                   │
│  CHAINCODES (Go)                                                 │
│  • did-registry-cc     : DID CRUD operations                    │
│  • credential-anchor-cc: Hash anchor + revocation               │
│  • audit-log-cc        : Anonymized verification events         │
│  • electoral-cc        : ZK-STARK proof anchoring               │
│                                                                   │
│  ENDORSEMENT POLICY                                              │
│  Credential operations: 3-of-5 NIA peer endorsement             │
│  DID registration: 2-of-3 NIA peer endorsement                  │
│  Revocation: 2-of-2 (NIA + issuing org) endorsement             │
│                                                                   │
│  PRIVATE DATA COLLECTIONS                                        │
│  Sensitive credential metadata shared only between              │
│  authorized organizations; hash on public ledger only           │
└─────────────────────────────────────────────────────────────────┘
```

## 8.3 Migration Strategy

If the initial blockchain selection needs to be changed (e.g., starting with a simpler implementation and migrating to Fabric later), the abstraction layer in Section 5.3 enables this without application-layer changes.

**Migration Steps:**
1. Deploy new blockchain network in parallel
2. Export all DID documents and credential anchors from existing chain
3. Replay all historical transactions on new chain (cryptographically verified)
4. Switch abstraction layer adapter via configuration change
5. Verify zero data loss through cryptographic reconciliation
6. Decommission old chain after 30-day verification period

---

# PART IX: Open Questions and Decisions Required
# بخش نهم: سؤالات باز و تصمیمات مورد نیاز

The following items require **policy decisions from the Transitional Government** before final technical implementation can proceed.

| # | Decision Required | Options | Impact on Timeline | Recommendation |
|---|-------------------|---------|-------------------|----------------|
| 1 | **Blockchain Platform Selection** | Hyperledger Fabric / Besu / Custom | ±3 months if delayed past Month 1 | Start Fabric evaluation immediately; select by end of Month 1 |
| 2 | **Biometric Sovereignty** | Fully onshore from Day 1 / International deduplication partner during Phase 1 | ±3–6 months if fully onshore required | Phased: partner for Phase 1, migrate onshore by Phase 3 |
| 3 | **Diaspora Voting Eligibility** | Eligible for referendum / Eligible for Mehestan elections only / Not eligible | Determines Phase 2 scope significantly | Recommended: Eligible for all; aligns with Emergency Phase Booklet |
| 4 | **Social Attestation Threshold** | 3 co-attestors / 5 co-attestors / Tiered by region | Minor development impact; major policy impact | 3 co-attestors minimum; 5 recommended for border regions |
| 5 | **Physical Card Fee** | Free for all / Means-tested free / Fee for all | Budget and rollout speed | Free for first issuance; fee for replacement |
| 6 | **International Audit Partners** | Estonia e-Governance Academy / UNDP / UN OCHA / Others | Must be decided before Phase 0 ends | Estonia + UNDP recommended |
| 7 | **Minority Language Launch Scope** | All 5 at launch / Persian + Kurdish + Azerbaijani at launch | ~2 months development difference | Persian + 3 major regional languages at launch |
| 8 | **Data Retention After Death** | 10 years / 25 years / Permanent | Database sizing and legal framework | 25 years recommended; aligns with transitional justice needs |
| 9 | **Private Sector Verifier Fees** | Fee per verification / Annual license / Free | Revenue model for system sustainability | Annual license; graduated by organization size |
| 10 | **ZK Trusted Setup** | National ceremony only / International multi-party | Legitimacy and security | International multi-party; include diaspora participants |

---

## Appendix A: Glossary / واژه‌نامه

| Term | فارسی | Definition |
|------|-------|------------|
| DID | شناسه غیرمتمرکز | Decentralized Identifier — W3C standard for sovereign digital identity |
| Verifiable Credential (VC) | مدرک تأییدپذیر | Cryptographically signed digital credential conforming to W3C VC Data Model |
| ZK-SNARK | — | Zero-Knowledge Succinct Non-interactive Argument of Knowledge |
| ZK-STARK | — | Zero-Knowledge Scalable Transparent Argument of Knowledge (post-quantum) |
| HSM | ماژول امنیت سخت‌افزاری | Hardware Security Module — tamper-resistant key storage |
| INDIS | سیستم هویت دیجیتال ملی ایران | Iran National Digital Identity System |
| NIA | سازمان هویت ملی | National Identity Authority — governing body for INDIS |
| RTL | راست‌به‌چپ | Right-to-Left text direction |
| Social Attestation | تأیید اجتماعی | Enrollment pathway for undocumented citizens using community co-attestors |
| Selective Disclosure | افشای انتخابی | Sharing only specific credential attributes, not the full credential |
| Liveness Detection | تشخیص زنده بودن | Biometric security preventing use of photos/videos to spoof enrollment |
| PAD | تشخیص حمله تقدیم | Presentation Attack Detection — ISO/IEC 30107-3 |

---

## Appendix B: Key Standards and References

| Standard | Scope | URL |
|----------|-------|-----|
| W3C DID Core 1.0 | Decentralized Identifiers | w3.org/TR/did-core |
| W3C VC Data Model 2.0 | Verifiable Credentials | w3.org/TR/vc-data-model-2.0 |
| OpenID4VP | Credential presentation | openid.net/specs/openid-4-verifiable-presentations |
| ISO/IEC 18013-5 | Mobile driving licence / digital ID | iso.org |
| ICAO 9303 | Physical travel document | icao.int |
| ISO/IEC 30107-3 | Biometric liveness / PAD | iso.org |
| ISO/IEC 29794 | Biometric sample quality | iso.org |
| FIPS 140-2 Level 3 | HSM security standard | nist.gov |
| NIST PQC (FIPS 203/204/205) | Post-quantum cryptography | nist.gov/pqcrypto |
| Circom 2.0 | ZK circuit language | docs.circom.io |
| Hyperledger Fabric | Permissioned blockchain | hyperledger-fabric.readthedocs.io |

---

## Appendix C: Revision History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 0.1 | ۱۴۰۵/۰۱ | Initial Draft | First complete draft |
| 1.0 | ۱۴۰۵/۰۱ | Review | Blockchain abstraction layer added; Hyperledger Fabric as candidate; Persian language requirements expanded; technology stack added |

---

*این سند نسخه ۱.۰ است و پس از بررسی ذینفعان به‌روزرسانی خواهد شد.*  
*This document represents Version 1.0 and will be updated following stakeholder review.*

*نسخه فارسی این سند برای اهداف حاکمیتی سند معتبر است.*  
*The Persian language version of this document is the authoritative version for governance purposes.*

*کد منبع تمام اجزای رمزنگاری باید متن‌باز و قابل ممیزی عمومی باشد.*  
*Source code for all cryptographic components must be open-source and publicly auditable.*

---

**IranProsperityProject.org | INDIS PRD v1.0 | ۱۴۰۵**
