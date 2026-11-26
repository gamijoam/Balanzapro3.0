import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/identity_catalog_main.dart";
import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  testWidgets("central typography has explicit hierarchy", (tester) async {
    BPVisuals? visuals;
    await tester.pumpWidget(
      BPApp(
        home: Builder(
          builder: (context) {
            visuals = BPVisuals.of(context);
            return const SizedBox.shrink();
          },
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(visuals!.heading.fontSize, 22);
    expect(visuals!.title.fontSize, 16);
    expect(visuals!.label.fontSize, 14);
    expect(visuals!.muted.fontSize, 13);
  });
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("navigation has its own semantic appearance $width", (
      tester,
    ) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      final nav = find.byKey(const ValueKey("bp-nav-products"));
      expect(
        find.descendant(
          of: nav,
          matching: find.byKey(const ValueKey("bp-nav-active")),
        ),
        findsOneWidget,
      );
      await tester.tap(find.byKey(const ValueKey("bp-nav-home")));
      await tester.pumpAndSettle();
      expect(
        find.descendant(
          of: nav,
          matching: find.byKey(const ValueKey("bp-nav-active")),
        ),
        findsNothing,
      );
      expect(find.text("Resumen de ejemplo"), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }

  test("semantic text and active navigation contrast", () {
    double ratio(Color a, Color b) {
      final x = a.computeLuminance(), y = b.computeLuminance();
      return x > y ? (x + .05) / (y + .05) : (y + .05) / (x + .05);
    }

    for (final (ink, muted, canvas, surface) in [
      (
        BPTokens.inkLight,
        BPTokens.mutedLight,
        BPTokens.canvasLight,
        const Color(0xFFFFFFFF),
      ),
      (
        BPTokens.inkDark,
        BPTokens.mutedDark,
        BPTokens.canvasDark,
        BPTokens.surfaceDark,
      ),
    ]) {
      for (final bg in [canvas, surface]) {
        expect(ratio(ink, bg), greaterThanOrEqualTo(4.5));
        expect(ratio(muted, bg), greaterThanOrEqualTo(4.5));
      }
    }
    for (final brightness in Brightness.values) {
      final (bg, fg) = BPTokens.navigationColors(brightness);
      expect(ratio(bg, fg), greaterThanOrEqualTo(4.5));
    }
  });
  testWidgets("identity catalogue with large dark text", (tester) async {
    tester.view.physicalSize = const Size(360, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    await tester.pumpWidget(const BPApp(home: BPIdentityExample()));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey("bp-nav-home")));
    await tester.pumpAndSettle();
    expect(
      find.descendant(
        of: find.byKey(const ValueKey("bp-nav-home")),
        matching: find.byKey(const ValueKey("bp-nav-active")),
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });
}
