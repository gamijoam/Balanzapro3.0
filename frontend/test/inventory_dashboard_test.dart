import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/design_system/bp_metrics.dart";
import "package:balanzapro_alpha/design_system/bp_badge.dart";
import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:balanzapro_alpha/metrics_catalog_main.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("dashboard composition and draft shortcut $width", (
      tester,
    ) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey("bp-nav-home")));
      await tester.pumpAndSettle();
      final scroll = find
          .descendant(
            of: find.byKey(const ValueKey("home")),
            matching: find.byType(Scrollable),
          )
          .first;
      await tester.scrollUntilVisible(
        find.text("Distribución de existencias").hitTestable(),
        100,
        scrollable: scroll,
      );
      await tester.pumpAndSettle();
      expect(find.text("Distribución de existencias"), findsOneWidget);
      expect(find.text("1 de 3 disponibles"), findsOneWidget);
      final shortcut = find.byKey(const ValueKey("dashboard-new"));

      await tester.scrollUntilVisible(
        shortcut.hitTestable(),
        -150,
        scrollable: scroll,
      );
      await tester.pumpAndSettle();
      await tester.tap(shortcut);
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(EditableText).first,
        "Producto temporal",
      );
      await tester.enterText(find.byType(EditableText).last, "TEMP-1");
      await tester.tap(find.byKey(const ValueKey("bp-nav-home")));
      await tester.pumpAndSettle();

      await tester.scrollUntilVisible(
        shortcut.hitTestable(),
        -150,
        scrollable: scroll,
      );
      await tester.pumpAndSettle();
      expect(find.text("Continuar borrador"), findsOneWidget);
      expect(
        tester
            .widget<BPButton>(find.byKey(const ValueKey("dashboard-low")))
            .onPressed,
        isNull,
      );
      await tester.tap(shortcut);
      await tester.pumpAndSettle();
      expect(find.text("Producto temporal"), findsOneWidget);
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey("bp-nav-home")));
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
        find.text("1 de 4 disponibles").hitTestable(),
        150,
        scrollable: scroll,
      );
      await tester.pumpAndSettle();
      expect(find.text("1 de 4 disponibles"), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }

  test("distribution rejects negative counts", () {
    expect(
      () => BPDistribution(
        title: "Ejemplo",
        parts: const [BPDistributionPart("disponibles", -1, BPTone.success)],
      ),
      throwsArgumentError,
    );
  });
  testWidgets("metrics catalog empty and scaled dark", (tester) async {
    tester.view.physicalSize = const Size(360, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    await tester.pumpWidget(const BPApp(home: BPMetricsExample()));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(
      find.text("Sin registros").hitTestable(),
      150,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();
    expect(find.text("0 de 0 disponibles"), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
