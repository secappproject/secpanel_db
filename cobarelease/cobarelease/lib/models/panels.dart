import 'dart:convert';

class Panel {
  String noPp;
  String noPanel;
  String noWbs;
  double? percentProgress;
  DateTime? startDate;
  DateTime? targetDelivery;
  String? statusBusbarPcc;
  String? statusBusbarMcc;
  String? statusComponent;
  String? statusPalet;
  String? statusCorepart;

  DateTime? aoBusbarPcc;
  DateTime? etaBusbarPcc;
  DateTime? aoBusbarMcc;
  DateTime? etaBusbarMcc;
  DateTime? aoComponent;
  DateTime? etaComponent;

  List<Map<String, dynamic>>? logs;
  String createdBy;
  String? vendorId;
  bool isClosed;
  DateTime? closedDate;

  Panel({
    required this.noPp,
    required this.noPanel,
    required this.noWbs,
    this.percentProgress,
    this.startDate,
    this.targetDelivery,
    this.statusBusbarPcc,
    this.statusBusbarMcc,
    this.statusComponent,
    this.statusCorepart,
    this.statusPalet,
    this.aoBusbarPcc,
    this.etaBusbarPcc,
    this.aoBusbarMcc,
    this.etaBusbarMcc,
    this.aoComponent,
    this.etaComponent,
    this.logs,
    required this.createdBy,
    this.vendorId,
    this.isClosed = false,
    this.closedDate,
  });

  Map<String, dynamic> toMap() {
    return {
      'no_pp': noPp,
      'no_panel': noPanel,
      'no_wbs': noWbs,
      'percent_progress': percentProgress,
      'start_date': startDate?.toIso8601String(),
      'target_delivery': targetDelivery?.toIso8601String(),
      'status_busbar_pcc': statusBusbarPcc,
      'status_busbar_mcc': statusBusbarMcc,
      'status_component': statusComponent,
      'status_palet': statusPalet,
      'status_corepart': statusCorepart,
      'ao_busbar_pcc': aoBusbarPcc?.toIso8601String(),
      'eta_busbar_pcc': etaBusbarPcc?.toIso8601String(),
      'ao_busbar_mcc': aoBusbarMcc?.toIso8601String(),
      'eta_busbar_mcc': etaBusbarMcc?.toIso8601String(),
      'ao_component': aoComponent?.toIso8601String(),
      'eta_component': etaComponent?.toIso8601String(),
      'logs': logs != null ? jsonEncode(logs) : null,
      'created_by': createdBy,
      'vendor_id': vendorId,
      'is_closed': isClosed ? 1 : 0,
      'closed_date': closedDate?.toIso8601String(),
    };
  }

  factory Panel.fromMap(Map<String, dynamic> map) {
    return Panel(
      noPp: map['no_pp'],
      noPanel: map['no_panel'],
      noWbs: map['no_wbs'],
      percentProgress: (map['percent_progress'] as num?)?.toDouble(),
      startDate: map['start_date'] != null
          ? DateTime.parse(map['start_date'])
          : null,
      targetDelivery: map['target_delivery'] != null
          ? DateTime.parse(map['target_delivery'])
          : null,
      statusBusbarPcc: map['status_busbar_pcc'],
      statusBusbarMcc: map['status_busbar_mcc'],
      statusComponent: map['status_component'],
      statusPalet: map['status_palet'],
      statusCorepart: map['status_corepart'],
      aoBusbarPcc: map['ao_busbar_pcc'] != null
          ? DateTime.parse(map['ao_busbar_pcc'])
          : null,
      etaBusbarPcc: map['eta_busbar_pcc'] != null
          ? DateTime.parse(map['eta_busbar_pcc'])
          : null,
      aoBusbarMcc: map['ao_busbar_mcc'] != null
          ? DateTime.parse(map['ao_busbar_mcc'])
          : null,
      etaBusbarMcc: map['eta_busbar_mcc'] != null
          ? DateTime.parse(map['eta_busbar_mcc'])
          : null,
      aoComponent: map['ao_component'] != null
          ? DateTime.parse(map['ao_component'])
          : null,
      etaComponent: map['eta_component'] != null
          ? DateTime.parse(map['eta_component'])
          : null,
      logs: map['logs'] != null
          ? (jsonDecode(map['logs']) as List)
                .map((item) => item as Map<String, dynamic>)
                .toList()
          : null,
      createdBy: map['created_by'],
      vendorId: map['vendor_id'],
      isClosed: map['is_closed'] == 1,
      closedDate: map['closed_date'] != null
          ? DateTime.parse(map['closed_date'])
          : null,
    );
  }
}
