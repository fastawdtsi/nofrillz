import Foundation

struct APIConfig {
  let baseURL: URL

  static func fromInfoPlist() -> APIConfig {
    if let rawValue = Bundle.main.object(forInfoDictionaryKey: "NOFRILLZ_API_BASE_URL") as? String {
      let trimmed = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
      if let url = URL(string: trimmed), !trimmed.isEmpty {
        return APIConfig(baseURL: url)
      }
    }

    return APIConfig(baseURL: URL(string: "https://api.nofrillz.com")!)
  }
}
