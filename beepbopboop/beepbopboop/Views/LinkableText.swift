import SwiftUI
import UIKit

struct LinkableText: UIViewRepresentable {
    let text: String
    let font: UIFont
    let textColor: UIColor

    init(_ text: String, font: UIFont = .preferredFont(forTextStyle: .body), textColor: UIColor = .label) {
        self.text = text
        self.font = font
        self.textColor = textColor
    }

    func makeUIView(context: Context) -> UITextView {
        let textView = UITextView()
        textView.isEditable = false
        textView.isScrollEnabled = false
        textView.backgroundColor = .clear
        textView.dataDetectorTypes = [.link, .phoneNumber]
        textView.textContainerInset = .zero
        textView.textContainer.lineFragmentPadding = 0
        textView.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        textView.setContentHuggingPriority(.defaultLow, for: .horizontal)
        textView.setContentHuggingPriority(.defaultHigh, for: .vertical)
        return textView
    }

    func updateUIView(_ textView: UITextView, context: Context) {
        textView.text = text
        textView.font = font
        textView.textColor = textColor
    }

    func sizeThatFits(_ proposal: ProposedViewSize, uiView: UITextView, context: Context) -> CGSize? {
        // Use proposed width when available; never exceed it to prevent overflow
        guard let proposedWidth = proposal.width, proposedWidth > 0, proposedWidth < .infinity else {
            // No valid width proposed — let SwiftUI figure it out
            let fallback = (uiView.window?.screen.bounds.width ?? 390) - 32
            let fittingSize = uiView.sizeThatFits(CGSize(width: fallback, height: CGFloat.greatestFiniteMagnitude))
            return CGSize(width: fallback, height: fittingSize.height)
        }
        let fittingSize = uiView.sizeThatFits(CGSize(width: proposedWidth, height: CGFloat.greatestFiniteMagnitude))
        return CGSize(width: proposedWidth, height: fittingSize.height)
    }
}
