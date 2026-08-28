import Foundation

enum LogReadPlan: Equatable {
    case append(from: UInt64)
    case replace

    var offset: UInt64 {
        switch self {
        case .append(let offset):
            return offset
        case .replace:
            return 0
        }
    }
}

struct LogStreamState {
    var position: UInt64 = 0
    private(set) var generation = 0
    private(set) var isClearing = false

    mutating func reset() {
        generation += 1
        position = 0
        isClearing = false
    }

    mutating func beginClear() -> Int {
        generation += 1
        isClearing = true
        return generation
    }

    @discardableResult
    mutating func finishClear(generation clearGeneration: Int) -> Bool {
        guard isClearing, clearGeneration == generation else { return false }
        position = 0
        isClearing = false
        return true
    }

    @discardableResult
    mutating func failClear(generation clearGeneration: Int) -> Bool {
        guard isClearing, clearGeneration == generation else { return false }
        isClearing = false
        return true
    }

    func canApply(_ readGeneration: Int) -> Bool {
        !isClearing && readGeneration == generation
    }

    func readPlan(forFileSize fileSize: UInt64) -> LogReadPlan? {
        guard !isClearing, fileSize != position else { return nil }
        return fileSize < position ? .replace : .append(from: position)
    }
}
