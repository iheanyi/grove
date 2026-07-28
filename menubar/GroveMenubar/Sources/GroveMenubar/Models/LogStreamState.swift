import Foundation

struct LogStreamState {
    var position: UInt64 = 0
    private(set) var generation = 0
    private(set) var isClearing = false

    mutating func reset() {
        generation += 1
        position = 0
        isClearing = false
    }

    mutating func beginClear() {
        generation += 1
        isClearing = true
    }

    mutating func finishClear() {
        position = 0
        isClearing = false
    }

    mutating func failClear() {
        isClearing = false
    }

    func canApply(_ readGeneration: Int) -> Bool {
        !isClearing && readGeneration == generation
    }

    func readOffset(forFileSize fileSize: UInt64) -> UInt64? {
        guard !isClearing, fileSize != position else { return nil }
        return fileSize < position ? 0 : position
    }
}
