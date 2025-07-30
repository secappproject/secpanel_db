import 'dart:convert';

class Panel {
  String noPp;
  String noPanel;
  String noWbs;
  double? percentProgress;
  DateTime? startDate;
  String? statusBusbar;
  String? statusComponent;
  String? statusPalet;
  String? statusCorepart;

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
    this.statusBusbar,
    this.statusComponent,
    this.statusCorepart,
    this.statusPalet,
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
      'status_busbar': statusBusbar,
      'status_component': statusComponent,
      'status_palet': statusPalet,
      'status_corepart': statusCorepart,
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
      statusBusbar: map['status_busbar'],
      statusComponent: map['status_component'],
      statusPalet: map['status_palet'],
      statusCorepart: map['status_corepart'],
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
