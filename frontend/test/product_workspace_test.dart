import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/design_system/bp_choice_bar.dart";
import "package:balanzapro_alpha/design_system/bp_table.dart";
import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:balanzapro_alpha/catalog_main.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("dashboard to filtered products and direct edit $width", (
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
      expect(find.byKey(const ValueKey("dashboard-low")), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey("dashboard-low")));
      await tester.pumpAndSettle();
      expect(find.text("Café 250 g"), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      await tester.enterText(find.byType(EditableText), "ALI-001");
      await tester.pumpAndSettle();
      expect(find.text("No hay resultados"), findsOneWidget);
      await tester.tap(find.text("Limpiar filtros"));
      await tester.pumpAndSettle();
      expect(find.text("Arroz 1 kg"), findsOneWidget);
      await tester.tap(find.byKey(const ValueKey("product-filter-low")));
      await tester.pumpAndSettle();
      expect(find.text("Café 250 g"), findsOneWidget);
      final edit = find.text("Editar Café 250 g");
      await tester.scrollUntilVisible(
        edit.hitTestable(),
        150,
        scrollable: find
            .descendant(
              of: find.byKey(const ValueKey("products")),
              matching: find.byType(Scrollable),
            )
            .first,
      );
      await tester.pumpAndSettle();
      await tester.tap(edit);
      await tester.pumpAndSettle();
      expect(find.text("Editar producto de ejemplo"), findsOneWidget);
      await tester.tap(find.text("Cancelar borrador"));
      await tester.pumpAndSettle();
      expect(find.text("Café 250 g"), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  test("choice IDs must be unique and selection valid", () {
    BPChoiceBar make(List<BPChoice> choices, String selected) => BPChoiceBar(
      label: "Ejemplo",
      choices: choices,
      selectedId: selected,
      onChanged: (_) {},
    );
    expect(
      () => make(const [BPChoice("a", "A"), BPChoice("a", "B")], "a"),
      throwsArgumentError,
    );
    expect(() => make(const [BPChoice("a", "A")], "b"), throwsArgumentError);
  });

  testWidgets("catalog choice works with enlarged text", (tester) async {
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
    final target = find.byKey(const ValueKey("catalog-filter-attention"));
    await tester.scrollUntilVisible(
      target.hitTestable(),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();
    expect(
      tester.getSize(target).height,
      greaterThanOrEqualTo(BPTokens.touchTarget),
    );
    await tester.tap(target);
    await tester.pumpAndSettle();
    expect(find.text("Filtro de ejemplo: attention"), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  test("table rejects missing or ambiguous row action", () {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    BPTable make(ValueChanged<String>? select, ValueChanged<String>? action) =>
        BPTable(
          columns: const ["Nombre"],
          rows: const [],
          page: 0,
          pageCount: 0,
          searchController: controller,
          onSearch: (_) {},
          onPage: (_) {},
          onSelect: select,
          onRowAction: action,
        );
    expect(() => make(null, null), throwsArgumentError);
    expect(() => make((_) {}, (_) {}), throwsArgumentError);
  });
}
