import SwiftUI

/// Displays a single W3C Verifiable Credential as a card.
///
/// Shows credential type, issuer, expiry in Solar Hijri, and a revocation badge
/// when `record.isRevoked` is true — mirroring Android's CredentialCardAdapter.
struct CredentialCardView: View {

    let record: CredentialRecord

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header row
            HStack {
                Image(systemName: iconName(for: record.credentialType))
                    .foregroundColor(record.isRevoked ? .red : .blue)
                    .font(.title3)

                Text(localizedType(record.credentialType))
                    .font(.headline)
                    .foregroundColor(.white)

                Spacer()

                StatusBadge(record: record)
            }

            Divider().background(Color.white.opacity(0.12))

            // Detail rows
            DetailRow(label: "صادرکننده", value: record.issuer)
            DetailRow(label: "صدور", value: PersianCalendar.formatISO(record.issuedAt))
            DetailRow(label: "انقضا", value: PersianCalendar.formatISO(record.expiresAt))
        }
        .padding(16)
        .background(Color.white.opacity(0.07))
        .cornerRadius(16)
        .overlay(
            RoundedRectangle(cornerRadius: 16)
                .stroke(record.isRevoked ? Color.red.opacity(0.4) : Color.clear, lineWidth: 1)
        )
    }

    private func iconName(for type: String) -> String {
        switch type.lowercased() {
        case let t where t.contains("national"):   return "creditcard.fill"
        case let t where t.contains("passport"):   return "doc.fill"
        case let t where t.contains("voter"):      return "checkmark.seal.fill"
        case let t where t.contains("driver"):     return "car.fill"
        default:                                   return "doc.badge.checkmark"
        }
    }

    private func localizedType(_ type: String) -> String {
        switch type {
        case "NationalIdCredential":  return "کارت ملی دیجیتال"
        case "PassportCredential":    return "گذرنامه"
        case "VoterCredential":       return "مدرک رأی‌دهی"
        case "DriverLicense":         return "گواهینامه رانندگی"
        default:                      return type
        }
    }
}

// MARK: — Sub-views

private struct StatusBadge: View {
    let record: CredentialRecord

    var body: some View {
        let (label, color): (String, Color) = {
            if record.isRevoked { return ("ابطال‌شده", .red) }
            if record.isExpired { return ("منقضی", .orange) }
            return ("معتبر", .green)
        }()
        Text(label)
            .font(.caption2).bold()
            .padding(.horizontal, 8).padding(.vertical, 3)
            .background(color.opacity(0.15))
            .foregroundColor(color)
            .cornerRadius(6)
    }
}

private struct DetailRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack {
            Text(label)
                .font(.caption)
                .foregroundColor(Color.white.opacity(0.5))
                .frame(width: 60, alignment: .leading)
            Text(value)
                .font(.caption)
                .foregroundColor(Color.white.opacity(0.8))
        }
    }
}
