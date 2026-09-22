import Foundation

extension Error {
  var isUnauthorizedAPIError: Bool {
    guard let apiError = self as? APIError else { return false }
    if case .unauthorized = apiError {
      return true
    }
    return false
  }
}
