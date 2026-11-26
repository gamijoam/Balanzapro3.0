import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/design_system/catalog/table_example.dart";
import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("product detail and read-only history $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(EditableText), "CAF");
      await tester.pumpAndSettle();
      final details = find.byKey(const ValueKey("bp-row-details-2"));
      expect(details, findsOneWidget);
      await tester.scrollUntilVisible(
        details.hitTestable(),
        150,
        scrollable: find
            .descendant(
              of: find.byKey(const ValueKey("products")),
              matching: find.byType(Scrollable),
            )
            .first,
      );
      await tester.pumpAndSettle();
      await tester.tap(details);
      await tester.pumpAndSettle();
      expect(find.text("Ficha del producto"), findsOneWidget);
      expect(find.text("Café 250 g"), findsOneWidget);
      await tester.tap(find.text("Consultar movimientos"));
      await tester.pumpAndSettle();
      expect(find.text("Historial de ejemplo"), findsOneWidget);
      expect(find.text("Café 250 g"), findsWidgets);
      expect(find.text("Arroz 1 kg"), findsNothing);
      await tester.tap(find.byKey(const ValueKey("movement-filter-out")));
      await tester.pumpAndSettle();
      final record = find.byKey(const ValueKey("bp-row-action-m4"));
      await tester.scrollUntilVisible(
        record.hitTestable(),
        150,
        scrollable: find
            .descendant(
              of: find.byKey(const ValueKey("movement-history")),
              matching: find.byType(Scrollable),
            )
            .first,
      );
      await tester.pumpAndSettle();
      await tester.tap(record);
      await tester.pumpAndSettle();
      expect(find.text("Detalle del movimiento"), findsOneWidget);
      expect(find.text("Salida de ejemplo"), findsOneWidget);
      await tester.tap(find.text("Volver al historial"));
      await tester.pumpAndSettle();
      await tester.tap(find.text("Volver al producto"));
      await tester.pumpAndSettle();
      expect(find.text("Ficha del producto"), findsOneWidget);
      await tester.tap(find.text("Volver a productos"));
      await tester.pumpAndSettle();
      expect(find.text("Café 250 g"), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets(
    "history follows stable product ID after rename, and empty search is recoverable",
    (tester) async {
      tester.view.physicalSize = const Size(1440, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(EditableText), "CAF");
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey("bp-row-action-2")));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(EditableText).first,
        "Café renombrado",
      );
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const ValueKey("bp-row-details-2")));
      await tester.pumpAndSettle();
      await tester.tap(find.text("Consultar movimientos"));
      await tester.pumpAndSettle();
      expect(find.text("Café renombrado"), findsWidgets);
      expect(find.text("Café 250 g"), findsNothing);
      await tester.enterText(find.byType(EditableText), "inexistente");
      await tester.pumpAndSettle();
      expect(find.text("No hay resultados"), findsOneWidget);
      await tester.tap(find.text("Limpiar filtros"));
      await tester.pumpAndSettle();
      expect(find.byKey(const ValueKey("bp-row-action-m4")), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      await tester.tap(find.byKey(const ValueKey("bp-nav-movements")));
      await tester.pumpAndSettle();
      expect(find.text("Arroz 1 kg"), findsWidgets);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets("catalog details action receives row ID without selecting", (
    tester,
  ) async {
    await tester.pumpWidget(const BPApp(home: BPTableExample()));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey("bp-row-details-demo-1")));
    await tester.pumpAndSettle();
    expect(find.text("Detalle de ejemplo: demo-1"), findsOneWidget);
    expect(find.text("Seleccionar Producto 1"), findsOneWidget);
  });
}
