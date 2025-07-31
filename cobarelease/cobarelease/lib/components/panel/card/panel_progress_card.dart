import 'package:flutter/material.dart';
import 'package:secpanel/components/alert_box.dart';
import 'package:secpanel/components/panel/card/remarks_bottom_sheet.dart';
import 'package:secpanel/theme/colors.dart';

class PanelProgressCard extends StatelessWidget {
  final String duration;
  final DateTime? targetDelivery;
  final double progress;
  final DateTime? startDate;
  final String progressLabel;
  final String panelTitle;
  final String statusBusbarPcc;
  final String statusBusbarMcc;
  final String statusComponent;
  final String statusPalet;
  final String statusCorepart;
  final String ppNumber;
  final String wbsNumber;
  final VoidCallback onEdit;
  final String panelVendorName;
  final String busbarVendorName;
  final String componentVendorName;
  final String paletVendorName;
  final String corepartVendorName;
  final bool isClosed;
  final DateTime? closedDate;
  final String? busbarRemarks;

  const PanelProgressCard({
    super.key,
    required this.duration,
    required this.targetDelivery,
    required this.progress,
    required this.startDate,
    required this.progressLabel,
    required this.panelTitle,
    required this.statusBusbarPcc,
    required this.statusBusbarMcc,
    required this.statusComponent,
    required this.statusPalet,
    required this.statusCorepart,
    required this.ppNumber,
    required this.wbsNumber,
    required this.onEdit,
    required this.panelVendorName,
    required this.busbarVendorName,
    required this.componentVendorName,
    required this.paletVendorName,
    required this.corepartVendorName,
    required this.isClosed,
    this.closedDate,
    this.busbarRemarks,
  });

  void _showRemarksBottomSheet(BuildContext context, String remarks) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => RemarksBottomSheet(remarks: remarks),
    );
  }

  int _extractDays(String durasi) {
    final hariMatch = RegExp(r'(\d+)\s*hari').firstMatch(durasi.toLowerCase());
    int hari = hariMatch != null ? int.parse(hariMatch.group(1)!) : 0;
    return hari;
  }

  // bool _shouldShowAlert() {
  //   return !isClosed && progress < 0.5 && _extractDays(duration) >= 2;
  // }

  bool _shouldShowAlert() {
    if (isClosed || targetDelivery == null) {
      return false; // Jangan tampilkan alert jika sudah ditutup atau tidak ada target
    }
    final now = DateTime.now();
    final difference = targetDelivery!.difference(now);

    // Tampilkan jika target belum lewat dan kurang dari 2 hari (48 jam) lagi
    return !difference.isNegative && difference.inHours < 48;
  }

  Color _getProgressColor() {
    if (isClosed) return AppColors.schneiderGreen;
    if (progress < 0.5) return AppColors.red; // Less than 50%
    if (progress < 0.75) return AppColors.orange; // 50% to 74.9%
    return AppColors.blue; // 75% to 100%
  }

  String _getProgressImage() {
    if (isClosed) return 'assets/images/progress-bolt-green.png';
    if (progress < 0.5)
      return 'assets/images/progress-bolt-red.png'; // Less than 50%
    if (progress < 0.75)
      return 'assets/images/progress-bolt-orange.png'; // 50% to 74.9%
    return 'assets/images/progress-bolt-blue.png'; // 75% to 100%
  }

  String _getBusbarStatusImage(String status) {
    final lower = status.toLowerCase();
    if (lower == 'n/a' || lower.contains('on progress')) {
      return 'assets/images/new-yellow.png';
    } else if (lower.contains('close')) {
      return 'assets/images/done-green.png';
    } else if (lower.contains('siap 100%')) {
      return 'assets/images/done-blue.png';
    } else if (lower.contains('red block')) {
      return 'assets/images/on-block-red.png';
    }
    return 'assets/images/no-status-gray.png';
  }

  String _getComponentStatusImage(String status) {
    final lower = status.toLowerCase();
    if (lower == 'n/a' || lower.contains('open')) {
      return 'assets/images/no-status-gray.png';
    } else if (lower.contains('done')) {
      return 'assets/images/done-green.png';
    } else if (lower.contains('on progress')) {
      return 'assets/images/on-progress-blue.png';
    }
    return 'assets/images/no-status-gray.png';
  }

  String _getPaletStatusImage(String status) {
    final lower = status.toLowerCase();
    if (lower == 'n/a' || lower.contains('open')) {
      return 'assets/images/no-status-gray.png';
    } else if (lower.contains('close')) {
      return 'assets/images/done-green.png';
    }
    return 'assets/images/no-status-gray.png';
  }

  String _getCorepartStatusImage(String status) {
    final lower = status.toLowerCase();
    if (lower == 'n/a' || lower.contains('open')) {
      return 'assets/images/no-status-gray.png';
    } else if (lower.contains('close')) {
      return 'assets/images/done-green.png';
    }
    return 'assets/images/no-status-gray.png';
  }

  String _formatTimeAgo(DateTime date) {
    final difference = DateTime.now().difference(date);
    if (difference.inDays > 0) {
      return '${difference.inDays} hari yang lalu';
    } else if (difference.inHours > 0) {
      return '${difference.inHours} jam yang lalu';
    } else if (difference.inMinutes > 0) {
      return '${difference.inMinutes} menit yang lalu';
    } else {
      return 'Baru saja';
    }
  }

  @override
  Widget build(BuildContext context) {
    final bool hasRemarks = busbarRemarks != null && busbarRemarks!.isNotEmpty;

    final String pccDisplayStatus = statusBusbarPcc ?? 'N/A';
    final String mccDisplayStatus = statusBusbarMcc ?? 'N/A';
    final String pccImageStatus = _getBusbarStatusImage(statusBusbarPcc);
    final String mccImageStatus = _getBusbarStatusImage(statusBusbarMcc);

    final String componentDisplayStatus = (statusComponent == 'N/A')
        ? 'Open'
        : statusComponent;
    final String paletDisplayStatus = (statusPalet == 'N/A')
        ? 'Open'
        : statusPalet;
    final String corepartDisplayStatus = (statusCorepart == 'N/A')
        ? 'Open'
        : statusCorepart;
    final bool isFuture =
        startDate != null && startDate!.isAfter(DateTime.now());
    final String durationLabel = isFuture ? "Mulai Dalam" : "Durasi Proses";

    return Column(
      children: [
        Container(
          decoration: BoxDecoration(
            borderRadius: const BorderRadius.all(Radius.circular(8)),
            border: Border.all(width: 1, color: AppColors.grayLight),
          ),
          width: MediaQuery.of(context).size.width,
          child: Column(
            children: [
              Container(
                padding: const EdgeInsets.all(12),
                width: double.infinity,
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Row(
                      children: [
                        Image.asset(_getProgressImage(), height: 28),
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.only(right: 8),
                          decoration: const BoxDecoration(
                            border: Border(
                              right: BorderSide(
                                color: AppColors.grayNeutral,
                                width: 1,
                              ),
                            ),
                          ),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                durationLabel,
                                style: TextStyle(
                                  color: AppColors.gray,
                                  fontWeight: FontWeight.w400,
                                  fontSize: 10,
                                ),
                              ),
                              Text(
                                duration,
                                style: const TextStyle(
                                  color: AppColors.black,
                                  fontWeight: FontWeight.w400,
                                  fontSize: 12,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Row(
                          children: [
                            // const Text(
                            //   "Panel",
                            //   style: TextStyle(
                            //     color: AppColors.gray,
                            //     fontWeight: FontWeight.w400,
                            //     fontSize: 10,
                            //   ),
                            // ),
                            // const SizedBox(width: 4),
                            // Container(
                            //   padding: const EdgeInsets.symmetric(
                            //     horizontal: 4,
                            //     vertical: 2,
                            //   ),
                            //   decoration: BoxDecoration(
                            //     color: AppColors.grayLight,
                            //     borderRadius: BorderRadius.circular(4),
                            //   ),
                            //   child: Text(
                            //     panelVendorName,
                            //     style: const TextStyle(
                            //       color: AppColors.black,
                            //       fontWeight: FontWeight.w400,
                            //       fontSize: 8,
                            //     ),
                            //   ),
                            // ),
                          ],
                        ),
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            Container(
                              height: 11,
                              width: MediaQuery.of(context).size.width - 256,
                              decoration: BoxDecoration(
                                color: Colors.grey[300],
                                borderRadius: BorderRadius.circular(20),
                              ),
                              child: FractionallySizedBox(
                                alignment: Alignment.centerLeft,
                                widthFactor: progress.clamp(0.0, 1.0),
                                child: Container(
                                  decoration: BoxDecoration(
                                    color: _getProgressColor(),
                                    borderRadius: BorderRadius.circular(20),
                                  ),
                                ),
                              ),
                            ),
                            const SizedBox(width: 12),
                            Text(
                              progressLabel,
                              style: const TextStyle(
                                color: AppColors.black,
                                fontWeight: FontWeight.w500,
                                fontSize: 12,
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: const BoxDecoration(
                  border: Border(
                    bottom: BorderSide(width: 1, color: AppColors.grayLight),
                    top: BorderSide(width: 1, color: AppColors.grayLight),
                  ),
                ),
                child: Row(
                  children: [
                    Container(
                      width: MediaQuery.of(context).size.width - 68,
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Container(
                                padding: EdgeInsets.all(8),
                                decoration: BoxDecoration(
                                  border: BoxBorder.all(
                                    width: 1,
                                    color: AppColors.grayLight,
                                  ),
                                  borderRadius: BorderRadius.all(
                                    Radius.circular(12),
                                  ),
                                ),
                                child: Row(
                                  mainAxisAlignment:
                                      MainAxisAlignment.spaceBetween,
                                  children: [
                                    Row(
                                      children: [
                                        const Text(
                                          "Panel",
                                          style: TextStyle(
                                            color: AppColors.gray,
                                            fontWeight: FontWeight.w400,
                                            fontSize: 10,
                                          ),
                                        ),
                                        const SizedBox(width: 4),
                                        Container(
                                          padding: const EdgeInsets.symmetric(
                                            horizontal: 4,
                                            vertical: 2,
                                          ),
                                          decoration: BoxDecoration(
                                            color: AppColors.grayLight,
                                            borderRadius: BorderRadius.circular(
                                              4,
                                            ),
                                          ),
                                          child: Text(
                                            panelVendorName == 'N/A'
                                                ? 'No Vendor'
                                                : panelVendorName,
                                            style: const TextStyle(
                                              color: AppColors.black,
                                              fontWeight: FontWeight.w400,
                                              fontSize: 10,
                                            ),
                                          ),
                                        ),
                                      ],
                                    ),
                                    Row(
                                      children: [
                                        const Text(
                                          "Busbar",
                                          style: TextStyle(
                                            color: AppColors.gray,
                                            fontWeight: FontWeight.w400,
                                            fontSize: 10,
                                          ),
                                        ),
                                        const SizedBox(width: 4),
                                        Container(
                                          padding: const EdgeInsets.symmetric(
                                            horizontal: 4,
                                            vertical: 2,
                                          ),
                                          decoration: BoxDecoration(
                                            color: AppColors.grayLight,
                                            borderRadius: BorderRadius.circular(
                                              4,
                                            ),
                                          ),
                                          child: Text(
                                            busbarVendorName == 'N/A'
                                                ? 'No Vendor'
                                                : busbarVendorName,
                                            style: const TextStyle(
                                              color: AppColors.black,
                                              fontWeight: FontWeight.w400,
                                              fontSize: 10,
                                            ),
                                          ),
                                        ),
                                        const SizedBox(width: 4),
                                        if (hasRemarks) ...[
                                          InkWell(
                                            onTap: () =>
                                                _showRemarksBottomSheet(
                                                  context,
                                                  busbarRemarks!,
                                                ),
                                            borderRadius: BorderRadius.circular(
                                              16,
                                            ),
                                            child: Container(
                                              padding: const EdgeInsets.all(4),
                                              decoration: BoxDecoration(
                                                borderRadius:
                                                    BorderRadius.circular(16),
                                                border: Border.all(
                                                  color: AppColors.grayLight,
                                                  width: 1,
                                                ),
                                              ),
                                              child: Image.asset(
                                                'assets/images/remarks.png',
                                                height: 16,
                                              ),
                                            ),
                                          ),
                                        ],
                                      ],
                                    ),
                                  ],
                                ),
                              ),
                              SizedBox(height: 12),
                              Text(
                                panelTitle,
                                style: const TextStyle(
                                  color: AppColors.black,
                                  fontWeight: FontWeight.w600,
                                  fontSize: 16,
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 12),
                          Column(
                            mainAxisAlignment: MainAxisAlignment.spaceBetween,
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Column(
                                children: [
                                  Row(
                                    children: [
                                      Expanded(
                                        child: Column(
                                          crossAxisAlignment:
                                              CrossAxisAlignment.start,
                                          children: [
                                            Row(
                                              children: [
                                                const Text(
                                                  "Busbar Pcc",
                                                  style: TextStyle(
                                                    color: AppColors.gray,
                                                    fontWeight: FontWeight.w400,
                                                    fontSize: 10,
                                                  ),
                                                ),
                                              ],
                                            ),
                                            const SizedBox(height: 4),
                                            Column(
                                              crossAxisAlignment:
                                                  CrossAxisAlignment.start,
                                              children: [
                                                Row(
                                                  children: [
                                                    Text(
                                                      pccDisplayStatus == 'N/A'
                                                          ? 'On Progress'
                                                          : pccDisplayStatus,
                                                      style: const TextStyle(
                                                        color: AppColors.black,
                                                        fontWeight:
                                                            FontWeight.w400,
                                                        fontSize: 10,
                                                      ),
                                                    ),
                                                    const SizedBox(width: 4),
                                                    Image.asset(
                                                      _getBusbarStatusImage(
                                                        statusBusbarPcc,
                                                      ),
                                                      height: 12,
                                                    ),
                                                  ],
                                                ),
                                              ],
                                            ),
                                          ],
                                        ),
                                      ),
                                      Expanded(
                                        child: Column(
                                          crossAxisAlignment:
                                              CrossAxisAlignment.start,
                                          children: [
                                            Row(
                                              children: [
                                                const Text(
                                                  "Busbar Mcc",
                                                  style: TextStyle(
                                                    color: AppColors.gray,
                                                    fontWeight: FontWeight.w400,
                                                    fontSize: 10,
                                                  ),
                                                ),
                                              ],
                                            ),
                                            const SizedBox(height: 4),
                                            Column(
                                              crossAxisAlignment:
                                                  CrossAxisAlignment.start,
                                              children: [
                                                Row(
                                                  children: [
                                                    Text(
                                                      mccDisplayStatus == 'N/A'
                                                          ? 'On Progress'
                                                          : mccDisplayStatus,
                                                      style: const TextStyle(
                                                        color: AppColors.black,
                                                        fontWeight:
                                                            FontWeight.w400,
                                                        fontSize: 10,
                                                      ),
                                                    ),
                                                    const SizedBox(width: 4),
                                                    Image.asset(
                                                      _getBusbarStatusImage(
                                                        statusBusbarMcc,
                                                      ),
                                                      height: 12,
                                                    ),
                                                  ],
                                                ),
                                              ],
                                            ),
                                          ],
                                        ),
                                      ),
                                      Container(
                                        alignment: Alignment.centerRight,
                                        width: 60,
                                        child: Container(
                                          alignment: Alignment.centerRight,
                                          child: Container(
                                            child: InkWell(
                                              onTap: onEdit,
                                              borderRadius:
                                                  BorderRadius.circular(8),
                                              child: Container(
                                                padding: const EdgeInsets.all(
                                                  8,
                                                ),
                                                decoration: BoxDecoration(
                                                  borderRadius:
                                                      BorderRadius.circular(8),
                                                  border: Border.all(
                                                    color: AppColors.grayLight,
                                                    width: 1,
                                                  ),
                                                ),
                                                child: Image.asset(
                                                  'assets/images/edit-green.png',
                                                  height: 20,
                                                ),
                                              ),
                                            ),
                                          ),
                                        ),
                                      ),
                                    ],
                                  ),
                                ],
                              ),
                              const SizedBox(height: 12),
                              Row(
                                children: [
                                  Expanded(
                                    child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        const Row(
                                          children: [
                                            Text(
                                              "Component",
                                              style: TextStyle(
                                                color: AppColors.gray,
                                                fontWeight: FontWeight.w400,
                                                fontSize: 10,
                                              ),
                                            ),
                                          ],
                                        ),
                                        const SizedBox(height: 4),
                                        Row(
                                          children: [
                                            Text(
                                              componentDisplayStatus,
                                              style: const TextStyle(
                                                color: AppColors.black,
                                                fontWeight: FontWeight.w400,
                                                fontSize: 10,
                                              ),
                                            ),
                                            const SizedBox(width: 4),
                                            Image.asset(
                                              _getComponentStatusImage(
                                                statusComponent,
                                              ),
                                              height: 12,
                                            ),
                                          ],
                                        ),
                                      ],
                                    ),
                                  ),
                                  Expanded(
                                    child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        const Row(
                                          children: [
                                            Text(
                                              "Palet",
                                              style: TextStyle(
                                                color: AppColors.gray,
                                                fontWeight: FontWeight.w400,
                                                fontSize: 10,
                                              ),
                                            ),
                                          ],
                                        ),
                                        const SizedBox(height: 4),
                                        Row(
                                          children: [
                                            Text(
                                              paletDisplayStatus,
                                              style: const TextStyle(
                                                color: AppColors.black,
                                                fontWeight: FontWeight.w400,
                                                fontSize: 10,
                                              ),
                                            ),
                                            const SizedBox(width: 4),
                                            Image.asset(
                                              _getPaletStatusImage(statusPalet),
                                              height: 12,
                                            ),
                                          ],
                                        ),
                                      ],
                                    ),
                                  ),
                                  Container(
                                    width: 60,
                                    child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      children: [
                                        const Row(
                                          children: [
                                            Text(
                                              "Corepart",
                                              style: TextStyle(
                                                color: AppColors.gray,
                                                fontWeight: FontWeight.w400,
                                                fontSize: 10,
                                              ),
                                            ),
                                          ],
                                        ),
                                        const SizedBox(height: 4),
                                        Row(
                                          children: [
                                            Text(
                                              corepartDisplayStatus,
                                              style: const TextStyle(
                                                color: AppColors.black,
                                                fontWeight: FontWeight.w400,
                                                fontSize: 10,
                                              ),
                                            ),
                                            const SizedBox(width: 4),
                                            Image.asset(
                                              _getCorepartStatusImage(
                                                statusCorepart,
                                              ),
                                              height: 12,
                                            ),
                                          ],
                                        ),
                                      ],
                                    ),
                                  ),
                                ],
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
              if (isClosed && closedDate != null)
                AlertBox(
                  title: "Closed",
                  description: _formatTimeAgo(closedDate!),
                  imagePath: 'assets/images/alert-success.png',
                  backgroundColor: const Color.fromARGB(11, 0, 138, 21),
                  borderColor: AppColors.schneiderGreen,
                  textColor: AppColors.schneiderGreen,
                )
              else if (_shouldShowAlert())
                AlertBox(
                  title: "Perlu Dikejar",
                  description: "Kurang dari 2 hari menuju target delivery!",
                ),
              Container(
                padding: const EdgeInsets.all(12),
                width: double.infinity,
                child: Column(
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Text(
                          "No. PP",
                          style: TextStyle(
                            color: AppColors.gray,
                            fontWeight: FontWeight.w300,
                            fontSize: 10,
                          ),
                        ),
                        Text(
                          ppNumber,
                          style: const TextStyle(
                            color: AppColors.gray,
                            fontWeight: FontWeight.w400,
                            fontSize: 10,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Text(
                          "No. WBS",
                          style: TextStyle(
                            color: AppColors.gray,
                            fontWeight: FontWeight.w300,
                            fontSize: 10,
                          ),
                        ),
                        Text(
                          wbsNumber,
                          style: const TextStyle(
                            color: AppColors.gray,
                            fontWeight: FontWeight.w400,
                            fontSize: 10,
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}
