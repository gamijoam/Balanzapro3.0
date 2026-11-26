import "package:balanzapro_alpha/main.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  testWidgets("live resize preserves search and draft through three modes", (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1440, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.pumpWidget(const BalanzaProApp());
    await tester.pumpAndSettle();
    final nav = find.byKey(const Key("desktop-navigation"));
    expect(tester.getTopLeft(nav).dy, lessThan(100));
    final toggle = find.byKey(const Key("bp-nav-toggle"));
    expect(toggle, findsOneWidget);
    final expanded = tester.getSize(nav).width;
    await tester.tap(toggle);
    await tester.pumpAndSettle();
    expect(tester.getSize(nav).width, lessThan(expanded));
    await tester.tap(toggle);
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(EditableText), "CAF");
    await tester.pumpAndSettle();
    for (final width in [1024.0, 1023.0, 800.0, 600.0, 599.0, 320.0, 1920.0]) {
      tester.view.physicalSize = Size(width, 900);
      await tester.pumpAndSettle();
      expect(
        find.byKey(
          Key(
            width < 600
                ? "compact-navigation"
                : width < 1024
                ? "rail-navigation"
                : "desktop-navigation",
          ),
        ),
        findsOneWidget,
      );
      expect(find.text("Café 250 g"), findsOneWidget);
      expect(find.text("Arroz 1 kg"), findsNothing);
      expect(tester.takeException(), isNull);
    }
    await tester.tap(find.text("Nuevo producto"));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(EditableText).first,
      "Borrador adaptable",
    );
    for (final width in [320.0, 800.0, 1440.0]) {
      tester.view.physicalSize = Size(width, 900);
      await tester.pumpAndSettle();
      expect(find.text("Borrador adaptable"), findsOneWidget);
      expect(tester.takeException(), isNull);
    }
  });
}
