import "package:balanzapro_alpha/main.dart";
import "package:flutter_test/flutter_test.dart";
import "package:flutter/widgets.dart";

void main() {
  for (final width in [360.0, 800.0, 1440.0]) {
    testWidgets("inventory adapts to width $width", (tester) async {
      tester.view.physicalSize = Size(width, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(const BalanzaProApp());
      await tester.pumpAndSettle();
      expect(
        find.text("Entorno de desarrollo · Datos ficticios"),
        findsOneWidget,
      );
      expect(find.text("Productos"), findsWidgets);
      expect(find.text("Arroz 1 kg"), findsOneWidget);
      expect(
        find.byKey(const Key("desktop-navigation")),
        width >= 1000 ? findsOneWidget : findsNothing,
      );
      expect(
        find.byKey(const Key("compact-navigation")),
        width < 1000 ? findsOneWidget : findsNothing,
      );
      await tester.enterText(find.byType(EditableText), "CAF");
      await tester.pumpAndSettle();
      expect(find.text("Café 250 g"), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      await tester.enterText(find.byType(EditableText), "no existe");
      await tester.pumpAndSettle();
      expect(find.text("No hay resultados"), findsOneWidget);
      await tester.tap(find.text("Limpiar búsqueda"));
      await tester.pumpAndSettle();
      expect(find.text("Arroz 1 kg"), findsOneWidget);
      await tester.tap(find.text("Movimientos"));
      await tester.pumpAndSettle();
      expect(
        find.text(
          "Los movimientos estarán disponibles al conectar la API protegida.",
        ),
        findsOneWidget,
      );
      await tester.tap(find.text("Productos"));
      await tester.pumpAndSettle();
      expect(find.text("Arroz 1 kg"), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  }
}
