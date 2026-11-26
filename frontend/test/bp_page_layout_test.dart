import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/design_system/bp_page_layout.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("shell keeps equivalent navigation at $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      expect(find.byKey(const Key("bp-page-layout")), findsOneWidget);
      await tester.tap(find.text("Inicio"));
      await tester.pumpAndSettle();
      expect(find.text("Resumen de ejemplo"), findsOneWidget);
      expect(
        find.text("Entorno de desarrollo · Datos ficticios"),
        findsOneWidget,
      );
      await tester.tap(find.text("Productos"));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(EditableText), "CAF");
      await tester.pumpAndSettle();
      await tester.tap(find.text("Inicio"));
      await tester.pumpAndSettle();
      await tester.tap(find.text("Productos"));
      await tester.pumpAndSettle();
      expect(find.text("Café 250 g"), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      tester.view.physicalSize = Size(width == 1440 ? 360 : 1440, 900);
      await tester.pumpAndSettle();
      expect(find.text("Café 250 g"), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets("compact shell on enlarged text and short landscape", (
    tester,
  ) async {
    tester.view.physicalSize = const Size(640, 360);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    await tester.pumpWidget(const BalanzaProApp());
    await tester.pumpAndSettle();
    expect(find.byKey(const Key("compact-navigation")), findsOneWidget);
    expect(find.text("Inicio"), findsOneWidget);
    await tester.tap(find.text("Inicio"));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });

  testWidgets("dashboard action returns to products with 200 percent text", (
    tester,
  ) async {
    tester.view.physicalSize = const Size(360, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    await tester.pumpWidget(const BalanzaProApp());
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey("bp-nav-home")));
    await tester.pumpAndSettle();
    final action = find.text("Ver productos");
    final scroll = find
        .descendant(
          of: find.byType(ListView),
          matching: find.byType(Scrollable),
        )
        .first;
    await tester.scrollUntilVisible(
      action.hitTestable(),
      180,
      scrollable: scroll,
    );
    await tester.pumpAndSettle();
    expect(action.hitTestable(), findsOneWidget);
    await tester.tap(action);
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey("products")), findsOneWidget);
    final field = find.byType(EditableText);
    await tester.scrollUntilVisible(
      field.hitTestable(),
      100,
      scrollable: find
          .descendant(
            of: find.byKey(const ValueKey("products")),
            matching: find.byType(Scrollable),
          )
          .first,
    );
    await tester.pumpAndSettle();
    expect(field.hitTestable(), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  test("rejects unknown selection and duplicate IDs", () {
    BPPageLayout layout(List<BPNavItem> items, String id) => BPPageLayout(
      title: "BP",
      contextLabel: "Ejemplo",
      items: items,
      selectedId: id,
      onNavigate: (_) {},
      child: const SizedBox.shrink(),
    );
    expect(
      () => layout(const [BPNavItem("a", "Uno")], "unknown"),
      throwsArgumentError,
    );
    expect(
      () => layout(const [BPNavItem("a", "Uno"), BPNavItem("a", "Dos")], "a"),
      throwsArgumentError,
    );
  });
}
