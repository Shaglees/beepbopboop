import SwiftUI

struct ContentMixView: View {
    @StateObject private var viewModel: ContentMixViewModel

    init(apiService: APIService) {
        _viewModel = StateObject(wrappedValue: ContentMixViewModel(apiService: apiService))
    }

    var body: some View {
        Section("Content Mix") {
            if viewModel.isLoading {
                ProgressView()
            } else {
                // Summary bar
                GeometryReader { geo in
                    HStack(spacing: 0) {
                        ForEach(ContentMixViewModel.verticalInfo, id: \.key) { info in
                            let weight = viewModel.targets[info.key] ?? 0
                            if weight > 0 {
                                Rectangle()
                                    .fill(Color(hexString: ContentMixViewModel.verticalColors[info.key] ?? "#888888"))
                                    .frame(width: geo.size.width * weight)
                            }
                        }
                    }
                    .clipShape(RoundedRectangle(cornerRadius: 4))
                }
                .frame(height: 8)
                .listRowBackground(Color.clear)

                // Vertical rows
                ForEach(ContentMixViewModel.verticalInfo, id: \.key) { info in
                    HStack {
                        Text(info.emoji)
                        Text(info.name)
                            .fontWeight(.medium)

                        if info.key == viewModel.omega {
                            Text("Ω")
                                .font(.caption2)
                                .fontWeight(.bold)
                                .foregroundColor(.white)
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.green)
                                .clipShape(Capsule())
                        }

                        Spacer()

                        // Status dot
                        let st = viewModel.status[info.key] ?? "on_target"
                        Circle()
                            .fill(st == "on_target" ? Color.green : st == "below_target" ? Color.orange : Color.blue)
                            .frame(width: 6, height: 6)

                        Text("\(Int((viewModel.targets[info.key] ?? 0) * 100))%")
                            .foregroundColor(.secondary)
                            .monospacedDigit()

                        Button {
                            viewModel.togglePin(info.key)
                        } label: {
                            Image(systemName: viewModel.pinned.contains(info.key) ? "pin.fill" : "pin")
                                .foregroundColor(viewModel.pinned.contains(info.key) ? .primary : .secondary.opacity(0.3))
                        }
                        .buttonStyle(.plain)
                    }
                }

                // Auto-adjust toggle
                Toggle("Auto-adjust from engagement", isOn: $viewModel.autoAdjust)
                    .onChange(of: viewModel.autoAdjust) { _, _ in
                        Task { await viewModel.save() }
                    }
            }
        }
        .task { await viewModel.load() }
    }
}
