import "package:flutter/widgets.dart";

/// Breakpoints refer to available logical width, never device identity.
enum BPLayoutMode { compact, rail, expanded }

abstract final class BPAdaptive {
  static const double compactLimit = 600, expandedLimit = 1024;
  static const double railWidth = 88, sidebarMin = 176, sidebarMax = 224;
  static const double expandedFraction = .19;
  static const double normalTextLimit = 21;

  static BPLayoutMode mode(double width, TextScaler scaler) {
    if (width < compactLimit || scaler.scale(14) > normalTextLimit) {
      return BPLayoutMode.compact;
    }
    return width < expandedLimit ? BPLayoutMode.rail : BPLayoutMode.expanded;
  }

  static double sideWidth(double available, bool collapsed) => collapsed
      ? railWidth
      : (available * expandedFraction).clamp(sidebarMin, sidebarMax);
}
