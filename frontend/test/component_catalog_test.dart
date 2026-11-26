import "package:balanzapro_alpha/catalog_main.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";
import "package:flutter/services.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("catalog works at width $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BPCatalogApp());
      await tester.pumpAndSettle();
      expect(find.text("Catálogo de componentes"), findsOneWidget);
      final primary = find.byKey(const Key("catalog-primary"));
      await tester.ensureVisible(primary);
      await tester.tap(primary);
      await tester.pumpAndSettle();
      expect(find.text("Acciones de ejemplo: 1"), findsOneWidget);
      final disabled = find.byKey(const Key("catalog-disabled"));
      await tester.ensureVisible(disabled);
      await tester.tap(disabled, warnIfMissed: false);
      await tester.pumpAndSettle();
      expect(find.text("Acciones de ejemplo: 1"), findsOneWidget);
      final loading = find.byKey(const Key("catalog-loading"));
      await tester.ensureVisible(loading);
      await tester.tap(loading);
      await tester.pumpAndSettle();
      expect(find.byKey(const Key("catalog-skeleton")), findsOneWidget);
      final loaded = find.byKey(const Key("catalog-loaded"));
      await tester.ensureVisible(loaded);
      await tester.tap(loaded);
      await tester.pumpAndSettle();
      expect(find.byKey(const Key("catalog-skeleton")), findsNothing);
      expect(find.text("Producto de ejemplo"), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets("catalog supports enlarged text and keyboard input", (
    tester,
  ) async {
    tester.view.physicalSize = const Size(360, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    await tester.pumpWidget(const BPCatalogApp());
    await tester.pumpAndSettle();
    final field = find.byType(EditableText);
    await tester.scrollUntilVisible(
      field.hitTestable(),
      180,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();
    expect(field.hitTestable(), findsOneWidget);
    await tester.tap(field);
    await tester.enterText(field, "ABC");
    await tester.sendKeyEvent(LogicalKeyboardKey.backspace);
    await tester.pumpAndSettle();
    expect(find.text("AB"), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
