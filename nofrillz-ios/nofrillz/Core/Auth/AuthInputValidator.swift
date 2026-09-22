import Foundation

enum AuthInputValidator {
  private static let usernameRegex = "^[A-Za-z0-9_-]+(?:\\.[A-Za-z0-9_-]+)*$"

  static func isValidEmail(_ value: String) -> Bool {
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty else { return false }

    let parts = trimmed.split(separator: "@", omittingEmptySubsequences: false)
    guard parts.count == 2 else { return false }

    let local = String(parts[0])
    let domain = String(parts[1])
    guard !local.isEmpty, !domain.isEmpty else { return false }
    guard domain.contains("."), !domain.hasPrefix("."), !domain.hasSuffix(".") else { return false }
    guard !trimmed.contains(" ") else { return false }

    let regex = "^[A-Z0-9a-z._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}$"
    return NSPredicate(format: "SELF MATCHES %@", regex).evaluate(with: trimmed)
  }

  static func isValidUsername(_ value: String) -> Bool {
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    guard (3 ... 30).contains(trimmed.count) else { return false }
    return NSPredicate(format: "SELF MATCHES %@", usernameRegex).evaluate(with: trimmed)
  }
}
