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
        textView.setContentHuggingPriority(.defaultHigh, for: .vertical)
        return textView
    }

    func updateUIView(_ textView: UITextView, context: Context) {
        textView.text = text
        textView.font = font
        textView.textColor = textColor
    }

    func sizeThatFits(_ proposal: ProposedViewSize, uiView: UITextView, context: Context) -> CGSize? {
        let width = proposal.width ?? UIScreen.main.bounds.width
        let fittingSize = uiView.sizeThatFits(CGSize(width: width, height: CGFloat.greatestFiniteMagnitude))
        return CGSize(width: width, height: fittingSize.height)
    }
}
