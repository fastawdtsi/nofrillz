import Foundation
import Network

enum ConnectivityStatus: Equatable {
  case unknown
  case online
  case offline
}

struct ConnectivityStatusChange {
  let previousStatus: ConnectivityStatus
  let currentStatus: ConnectivityStatus
}

@MainActor
final class ConnectivityMonitor {
  private let monitor = NWPathMonitor()
  private let queue = DispatchQueue(label: "com.rosso-technologies.nofrillz.connectivity")
  private var isStarted = false
  private var pendingUpdateWorkItem: DispatchWorkItem?

  private(set) var status: ConnectivityStatus = .unknown

  var isOffline: Bool {
    status == .offline
  }

  func start() {
    guard !isStarted else { return }
    isStarted = true

    monitor.pathUpdateHandler = { [weak self] path in
      let nextStatus: ConnectivityStatus = path.status == .satisfied ? .online : .offline
      Task { @MainActor [weak self] in
        self?.scheduleStatusUpdate(nextStatus)
      }
    }
    monitor.start(queue: queue)
  }

  deinit {
    monitor.cancel()
  }

  private func scheduleStatusUpdate(_ nextStatus: ConnectivityStatus) {
    guard nextStatus != status else { return }

    pendingUpdateWorkItem?.cancel()

    let delay: TimeInterval
    switch nextStatus {
    case .offline:
      delay = 0.35
    case .online:
      delay = 0.65
    case .unknown:
      delay = 0
    }

    let workItem = DispatchWorkItem { [weak self] in
      self?.applyStatus(nextStatus)
    }
    pendingUpdateWorkItem = workItem
    DispatchQueue.main.asyncAfter(deadline: .now() + delay, execute: workItem)
  }

  private func applyStatus(_ nextStatus: ConnectivityStatus) {
    pendingUpdateWorkItem = nil
    guard nextStatus != status else { return }

    let previousStatus = status
    status = nextStatus

    NotificationCenter.default.post(
      name: .connectivityStatusDidChange,
      object: ConnectivityStatusChange(previousStatus: previousStatus, currentStatus: nextStatus)
    )
  }
}
