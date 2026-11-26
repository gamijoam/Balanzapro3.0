import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/design_system/bp_table.dart";
import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  testWidgets("continuous table and compact actions in wide content", (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1440, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(const BalanzaProApp());
    await tester.pumpAndSettle();
    final surface = find.byKey(const ValueKey("bp-data-surface"));
    expect(surface, findsOneWidget);
    final row = find.byKey(const ValueKey("bp-data-row-1"));
    expect(row, findsOneWidget);
    expect(tester.getSize(row).height, lessThanOrEqualTo(72));
    final edit = find.byKey(const ValueKey("bp-row-action-1"));
    expect(tester.getSize(edit).width, lessThanOrEqualTo(64));
    expect(tester.getSize(edit).height, greaterThanOrEqualTo(48));
    await tester.enterText(find.byType(EditableText), "CAF");
    await tester.pumpAndSettle();
    for (final width in [800.0, 320.0, 1440.0]) {
      tester.view.physicalSize = Size(width, 900);
      await tester.pumpAndSettle();
      expect(
        find.byKey(
          ValueKey(width == 1440 ? "bp-table-wide" : "bp-table-compact"),
        ),
        findsOneWidget,
      );
      expect(find.text("Café 250 g"), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      expect(tester.takeException(), isNull);
    }
    await tester.tap(find.byKey(const ValueKey("bp-row-action-2")));
    await tester.pumpAndSettle();
    expect(find.text("Editar producto de ejemplo"), findsOneWidget);
  });

  testWidgets(
    "long records and contextual actions survive resize and enlarged text",
    (tester) async {
      tester.view.physicalSize = const Size(1440, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final dispatcher = tester.binding.platformDispatcher;
      addTearDown(dispatcher.clearTextScaleFactorTestValue);
      final controller = TextEditingController();
      addTearDown(controller.dispose);
      final name = List.filled(12, "Producto largo").join(" ");
      String? action;
      await tester.pumpWidget(
        BPApp(
          home: BPTable(
            columns: const ["Producto", "Código", "Stock", "Estado"],
            rows: [
              BPTableRow("long", [
                name,
                "COD-12345678901234567890",
                "100",
                "Disponible",
              ]),
            ],
            page: 0,
            pageCount: 1,
            searchController: controller,
            onSearch: (_) {},
            onPage: (_) {},
            rowActionLabel: "Editar",
            onRowAction: (id) => action = id,
          ),
        ),
      );
      await tester.pumpAndSettle();
      final semantics = tester.ensureSemantics();

      expect(find.text(name), findsOneWidget);
      expect(find.bySemanticsLabel("Editar $name"), findsWidgets);
      await tester.tap(find.byKey(const ValueKey("bp-row-action-long")));
      expect(action, "long");
      tester.view.physicalSize = const Size(320, 800);
      dispatcher.textScaleFactorTestValue = 2;
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey("bp-table-compact")), findsOneWidget);
      expect(find.text(name), findsOneWidget);
      expect(tester.takeException(), isNull);
      semantics.dispose();
    },
  );
}
