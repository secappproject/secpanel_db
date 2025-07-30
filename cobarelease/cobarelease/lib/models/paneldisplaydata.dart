import 'package:secpanel/models/panels.dart';

class PanelDisplayData {
  final Panel panel;
  final String panelVendorName;
  final String busbarVendorNames;
  final List<String> busbarVendorIds;
  final String componentVendorNames;
  final List<String> componentVendorIds;
  final String? busbarRemarks;

  PanelDisplayData({
    required this.panel,
    required this.panelVendorName,
    required this.busbarVendorNames,
    required this.busbarVendorIds,
    required this.componentVendorNames,
    required this.componentVendorIds,
    this.busbarRemarks,
  });
}
