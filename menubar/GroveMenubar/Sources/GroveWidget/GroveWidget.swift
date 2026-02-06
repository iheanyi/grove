import SwiftUI
import WidgetKit

/// The main widget definition supporting small, medium, and large sizes.
struct GroveServerWidget: Widget {
    let kind = "com.grove.server-status"

    var body: some WidgetConfiguration {
        StaticConfiguration(kind: kind, provider: ServerStatusProvider()) { entry in
            GroveWidgetEntryView(entry: entry)
        }
        .configurationDisplayName("Grove Servers")
        .description("Monitor your development servers at a glance.")
        .supportedFamilies([.systemSmall, .systemMedium, .systemLarge])
    }
}

/// Dispatches to the correct view based on widget family.
struct GroveWidgetEntryView: View {
    @Environment(\.widgetFamily) var family
    let entry: ServerEntry

    var body: some View {
        switch family {
        case .systemSmall:
            SmallWidgetView(entry: entry)
        case .systemMedium:
            MediumWidgetView(entry: entry)
        case .systemLarge:
            LargeWidgetView(entry: entry)
        default:
            MediumWidgetView(entry: entry)
        }
    }
}

// MARK: - Previews

#if DEBUG
#Preview("Small", as: .systemSmall) {
    GroveServerWidget()
} timeline: {
    ServerEntry.placeholder
    ServerEntry(date: .now, servers: [])
}

#Preview("Medium", as: .systemMedium) {
    GroveServerWidget()
} timeline: {
    ServerEntry.placeholder
}

#Preview("Large", as: .systemLarge) {
    GroveServerWidget()
} timeline: {
    ServerEntry.placeholder
}
#endif
