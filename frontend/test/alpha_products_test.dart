import "package:balanzapro_alpha/main.dart";
import "package:balanzapro_alpha/design_system/bp_table.dart";
import "package:balanzapro_alpha/design_system/bp_form.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("alpha product create and edit $width", (tester) async {
      tester.view.physicalSize = Size(width, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      expect(find.byType(BPTable), findsOneWidget);
      await tester.tap(find.text("Nuevo producto"));
      await tester.pumpAndSettle();
      expect(find.byType(BPForm), findsOneWidget);
      await tester.enterText(
        find.byType(EditableText).first,
        "Producto de prueba",
      );
      await tester.enterText(find.byType(EditableText).last, "ALI-001");
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      expect(find.text("Ese código ya existe en el ejemplo"), findsOneWidget);
      await tester.enterText(find.byType(EditableText).last, "PRUEBA-1");
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      expect(find.byType(BPTable), findsOneWidget);
      await tester.enterText(find.byType(EditableText), "PRUEBA-1");
      await tester.pumpAndSettle();
      expect(find.text("Producto de prueba"), findsOneWidget);
      await tester.scrollUntilVisible(
        find.text("Editar Producto de prueba").hitTestable(),
        150,
        scrollable: find
            .descendant(
              of: find.byKey(const ValueKey("products")),
              matching: find.byType(Scrollable),
            )
            .first,
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text("Editar Producto de prueba"));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byType(EditableText).first,
        "Producto editado",
      );
      await tester.tap(find.text("Guardar"));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(EditableText), "PRUEBA-1");
      await tester.pumpAndSettle();
      expect(find.text("Producto editado"), findsOneWidget);
      expect(find.text("Producto de prueba"), findsNothing);
      expect(tester.takeException(), isNull);
    });
  }

  testWidgets("cancel draft does not add a product", (tester) async {
    tester.view.physicalSize = const Size(800, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(const BalanzaProApp());
    await tester.pumpAndSettle();
    await tester.tap(find.text("Nuevo producto"));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(EditableText).first,
      "Borrador descartado",
    );
    await tester.enterText(find.byType(EditableText).last, "DRAFT-1");
    await tester.tap(find.text("Cancelar borrador"));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(EditableText), "DRAFT-1");
    await tester.pumpAndSettle();
    expect(find.text("No hay resultados"), findsOneWidget);
    expect(find.text("Borrador descartado"), findsNothing);
    expect(tester.takeException(), isNull);
  });
}
