import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/design_system/bp_badge.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("navigation and status hierarchy $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      final nav = find.byKey(
        Key(
          width >= 1024
              ? "desktop-navigation"
              : width >= 600
              ? "rail-navigation"
              : "compact-navigation",
        ),
      );
      expect(nav, findsOneWidget);
      if (width < 600) {
        expect(tester.getTopLeft(nav).dy, greaterThan(650));
      }
      expect(find.byKey(const ValueKey("bp-status-1")), findsOneWidget);
      expect(find.text("Disponible"), findsOneWidget);
      expect(find.text("Stock bajo"), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey("bp-nav-home")));
      await tester.pumpAndSettle();
      expect(find.text("Resumen de ejemplo"), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }

  test("badge contrast AA in both themes", () {
    for (final tone in BPTone.values) {
      for (final brightness in Brightness.values) {
        final (bg, fg) = BPBadge.colors(tone, brightness);
        final a = bg.computeLuminance(), b = fg.computeLuminance();
        final ratio = a > b ? (a + .05) / (b + .05) : (b + .05) / (a + .05);
        expect(ratio, greaterThanOrEqualTo(4.5), reason: "$tone $brightness");
      }
    }
  });
}
