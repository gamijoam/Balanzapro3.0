import "package:balanzapro_alpha/design_system/design_system.dart";
import "package:flutter/widgets.dart";
import "package:flutter_test/flutter_test.dart";

void main() {
  testWidgets("shared buttons and skeleton consume central tokens", (
    tester,
  ) async {
    var presses = 0;
    await tester.pumpWidget(
      BPApp(
        home: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              BPButton(label: "Guardar", onPressed: () => presses++),
              const BPButton(label: "Eliminar", onPressed: null),
              const BPSkeleton(width: 120, height: 20),
            ],
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(
      tester.getSize(find.widgetWithText(BPButton, "Guardar")).height,
      BPTokens.touchTarget,
    );
    expect(
      tester.getSize(find.widgetWithText(BPButton, "Eliminar")).height,
      BPTokens.touchTarget,
    );
    expect(tester.getSize(find.byType(BPSkeleton)), const Size(120, 20));
    await tester.tap(find.text("Guardar"));
    await tester.tap(find.text("Eliminar"), warnIfMissed: false);
    expect(presses, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets("system theme switches semantic background", (tester) async {
    final dispatcher = tester.binding.platformDispatcher;
    dispatcher.platformBrightnessTestValue = Brightness.light;
    addTearDown(dispatcher.clearPlatformBrightnessTestValue);
    Color? background;
    await tester.pumpWidget(
      BPApp(
        home: Builder(
          builder: (context) {
            background = BPVisuals.of(context).background;
            return const SizedBox.shrink();
          },
        ),
      ),
    );
    await tester.pumpAndSettle();
    final light = background;
    expect(light, isNotNull);
    dispatcher.platformBrightnessTestValue = Brightness.dark;
    await tester.pumpAndSettle();
    expect(background, isNot(light));
    expect(background!.computeLuminance(), lessThan(light!.computeLuminance()));
  });

  test("primary token text contrast is at least AA", () {
    double contrast(Color a, Color b) {
      final x = a.computeLuminance(), y = b.computeLuminance();
      return x > y ? (x + .05) / (y + .05) : (y + .05) / (x + .05);
    }

    expect(
      contrast(BPTokens.primaryLight, BPTokens.onPrimaryLight),
      greaterThanOrEqualTo(4.5),
    );
    expect(
      contrast(BPTokens.primaryDark, BPTokens.onPrimaryDark),
      greaterThanOrEqualTo(4.5),
    );
  });
}
