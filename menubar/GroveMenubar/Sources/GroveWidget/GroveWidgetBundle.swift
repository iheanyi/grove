import SwiftUI
import WidgetKit

/// Entry point for the widget extension.
/// When built as a proper widget extension target in Xcode, this is the main
/// entry point that registers all Grove widgets.
@main
struct GroveWidgetBundle: WidgetBundle {
    var body: some Widget {
        GroveServerWidget()
    }
}
