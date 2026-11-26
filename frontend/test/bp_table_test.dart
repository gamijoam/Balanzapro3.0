import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:balanzapro_alpha/design_system/bp_table.dart";
import "package:flutter/widgets.dart";
import "package:balanzapro_alpha/design_system/catalog/table_example.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("table adapts and requests pages at $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final controller = TextEditingController();
      addTearDown(controller.dispose);
      int? next;
      String? selected, query;
      await tester.pumpWidget(
        BPApp(
          home: BPTable(
            columns: const ["Producto", "Código"],
            rows: const [
              BPTableRow("a", ["Arroz", "ALI-001"]),
              BPTableRow("b", ["Café", "ALI-002"]),
            ],
            page: 0,
            pageCount: 2,
            searchController: controller,
            onSearch: (v) => query = v,
            selectedId: null,
            onSelect: (v) => selected = v,
            onPage: (v) => next = v,
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        find.byKey(Key(width >= 1000 ? "bp-table-wide" : "bp-table-compact")),
        findsOneWidget,
      );
      await tester.tap(find.text("Seleccionar Arroz"));
      await tester.pumpAndSettle();
      expect(selected, "a");
      await tester.enterText(find.byType(EditableText), "CAF");
      expect(query, "CAF");
      await tester.tap(find.text("Siguiente"));
      await tester.pumpAndSettle();
      expect(next, 1);
      await tester.tap(find.text("Anterior"), warnIfMissed: false);
      expect(next, 1);
      expect(tester.takeException(), isNull);
    });
  }
  testWidgets("table states never display stale rows", (tester) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    var retries = 0;
    for (final state in BPTableState.values) {
      await tester.pumpWidget(
        BPApp(
          home: BPTable(
            columns: const ["Producto"],
            rows: state == BPTableState.ready
                ? const []
                : const [
                    BPTableRow("old", ["Registro anterior"]),
                  ],
            page: 0,
            pageCount: state == BPTableState.ready ? 0 : 1,
            searchController: controller,
            onSearch: (_) {},
            selectedId: null,
            onSelect: (_) {},
            onPage: (_) {},
            state: state,
            onRetry: () => retries++,
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text("Registro anterior"), findsNothing);
      if (state == BPTableState.loading) {
        expect(find.byType(BPSkeleton), findsWidgets);
      }
      if (state == BPTableState.ready) {
        expect(find.text("No hay resultados"), findsOneWidget);
      }
      if (state == BPTableState.error) {
        await tester.tap(find.text("Reintentar"));
        expect(retries, 1);
      }
      expect(tester.takeException(), isNull);
    }
  });

  testWidgets("example owns selection, pages and filtering", (tester) async {
    await tester.pumpWidget(const BPApp(home: BPTableExample()));
    await tester.pumpAndSettle();
    await tester.tap(find.text("Seleccionar Producto 1"));
    await tester.pumpAndSettle();
    expect(find.text("Seleccionado Producto 1"), findsOneWidget);
    await tester.scrollUntilVisible(
      find.text("Siguiente"),
      250,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.ensureVisible(find.text("Siguiente"));
    await tester.tap(find.text("Siguiente"));
    await tester.pumpAndSettle();
    expect(find.text("Página 2 de 3"), findsOneWidget);
    expect(find.text("Producto 1"), findsNothing);
    await tester.scrollUntilVisible(
      find.byType(EditableText).hitTestable(),
      -250,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.enterText(find.byType(EditableText), "SKU-9");
    await tester.pumpAndSettle();
    expect(find.text("Producto 9"), findsOneWidget);
    expect(find.text("Página 1 de 1"), findsOneWidget);
  });
  test("rejects invalid page contracts", () {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    BPTable make(List<BPTableRow> rows) => BPTable(
      columns: const ["Producto"],
      rows: rows,
      page: 0,
      pageCount: 1,
      searchController: controller,
      onSearch: (_) {},
      selectedId: null,
      onSelect: (_) {},
      onPage: (_) {},
    );
    expect(
      () => make(List.generate(51, (i) => BPTableRow("$i", ["x"]))),
      throwsArgumentError,
    );
    expect(() => make(const [BPTableRow("x", [])]), throwsArgumentError);
    expect(
      () => make(const [
        BPTableRow("x", ["a"]),
        BPTableRow("x", ["b"]),
      ]),
      throwsArgumentError,
    );
  });

  testWidgets("large text uses compact table and remains scrollable", (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1440, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.textScaleFactorTestValue = 2;
    addTearDown(dispatcher.clearTextScaleFactorTestValue);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    await tester.pumpWidget(const BPApp(home: BPTableExample()));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key("bp-table-compact")), findsOneWidget);
    expect(find.text("Producto 1"), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
