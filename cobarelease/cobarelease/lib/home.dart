import 'package:flutter/material.dart';
import 'package:shimmer/shimmer.dart';
import 'package:secpanel/components/panel/edit/edit_panel_bottom_sheet.dart';
import 'package:secpanel/components/panel/edit/edit_status_bottom_sheet.dart';
import 'package:secpanel/components/panel/filtersearch/panel_filter_bottom_sheet.dart';
import 'package:secpanel/components/panel/card/panel_progress_card.dart';
import 'package:secpanel/components/panel/filtersearch/search_field.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/paneldisplaydata.dart';
import 'package:secpanel/models/panels.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';

class HomeScreen extends StatefulWidget {
  final Company currentCompany;
  final VoidCallback onRefresh;

  const HomeScreen({
    super.key,
    required this.currentCompany,
    required this.onRefresh,
  });

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> with TickerProviderStateMixin {
  late final TabController _tabController;
  final TextEditingController _searchController = TextEditingController();

  List<PanelDisplayData> _allPanelsData = [];
  List<Company> _allK3Vendors = [];
  List<Company> _allK5Vendors = [];
  List<Company> _allWHSVendors = [];
  bool _isLoading = true;

  String searchQuery = "";
  bool includeArchived = false;
  SortOption? selectedSort;
  List<PanelFilterStatus> selectedPanelStatuses = [];
  List<String> selectedPanelVendors = [];
  List<String> selectedBusbarVendors = [];
  List<String> selectedComponentVendors = [];
  List<String> selectedPaletVendors = [];
  List<String> selectedCorepartVendors = [];
  List<String> selectedStatuses = [];
  List<String> selectedComponents = [];
  List<String> selectedPalet = [];
  List<String> selectedCorepart = [];

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 5, vsync: this);
    _tabController.addListener(() => setState(() {}));
    _loadInitialData();
  }

  Future<void> _loadInitialData() async {
    setState(() {
      _isLoading = true;
    });
    await Future.delayed(const Duration(milliseconds: 500));

    final panelsDataFromDb = await DatabaseHelper.instance
        .getAllPanelsForDisplay(widget.currentCompany);

    final k3Vendors = await DatabaseHelper.instance.getK3Vendors();
    final k5Vendors = await DatabaseHelper.instance.getK5Vendors();
    final whsVendors = await DatabaseHelper.instance.getWHSVendors();

    if (mounted) {
      setState(() {
        _allPanelsData = panelsDataFromDb;
        _allK3Vendors = k3Vendors;
        _allK5Vendors = k5Vendors;
        _allWHSVendors = whsVendors;
        _isLoading = false;
      });
    }
  }

  @override
  void dispose() {
    _tabController.dispose();
    _searchController.dispose();
    super.dispose();
  }

  String _formatDuration(DateTime? startDate) {
    if (startDate == null) return "N/A";
    final now = DateTime.now();

    if (startDate.isAfter(now)) {
      final futureDifference = startDate.difference(now);

      final days = futureDifference.inDays;
      final hours = futureDifference.inHours % 24;
      final minutes = futureDifference.inMinutes % 60;

      return "$days hari $hours jam";
    } else {
      final pastDifference = now.difference(startDate);
      final days = pastDifference.inDays;
      final hours = pastDifference.inHours % 24;

      if (days == 0 && hours == 0) {
        return "Baru saja dimulai";
      }
      return "$days hari $hours jam";
    }
  }

  PanelFilterStatus _getPanelFilterStatus(Panel panel) {
    final progress = panel.percentProgress ?? 0;
    if (progress < 50) return PanelFilterStatus.progressRed;
    if (progress < 75) return PanelFilterStatus.progressOrange;
    if (progress < 100) return PanelFilterStatus.progressBlue;
    if (progress >= 100) {
      if (!panel.isClosed) {
        return PanelFilterStatus.readyToDelivery;
      } else {
        if (panel.closedDate != null &&
            DateTime.now().difference(panel.closedDate!).inHours > 48) {
          return PanelFilterStatus.closedArchived;
        } else {
          return PanelFilterStatus.closed;
        }
      }
    }
    return PanelFilterStatus.progressRed;
  }

  // Lokasi: file HomeScreen.dart

  List<PanelDisplayData> get _panelsAfterPrimaryFilters {
    return _allPanelsData.where((data) {
      final panel = data.panel;

      // --- Filter dasar (pencarian, vendor, dll.) ---
      final query = searchQuery.toLowerCase();
      final matchSearch =
          panel.noPanel.toLowerCase().contains(query) ||
          panel.noPp.toLowerCase().contains(query) ||
          panel.noWbs.toLowerCase().contains(query) ||
          data.panelVendorName.toLowerCase().contains(query) ||
          data.busbarVendorNames.toLowerCase().contains(query) ||
          data.componentVendorNames.toLowerCase().contains(query) ||
          data.paletVendorNames.toLowerCase().contains(query) ||
          data.corepartVendorNames.toLowerCase().contains(query);

      final matchPanelVendor =
          selectedPanelVendors.isEmpty ||
          selectedPanelVendors.contains(panel.vendorId);

      final matchBusbarVendor =
          selectedBusbarVendors.isEmpty ||
          selectedBusbarVendors.any((id) => data.busbarVendorIds.contains(id));

      final matchComponentVendor =
          selectedComponentVendors.isEmpty ||
          selectedComponentVendors.any(
            (id) => data.componentVendorIds.contains(id),
          );

      final matchPaletVendor =
          selectedPaletVendors.isEmpty ||
          selectedPaletVendors.any((id) => data.paletVendorIds.contains(id));

      final matchCorepartVendor =
          selectedCorepartVendors.isEmpty ||
          selectedCorepartVendors.any(
            (id) => data.corepartVendorIds.contains(id),
          );

      final matchStatus =
          selectedStatuses.isEmpty ||
          selectedStatuses.contains(panel.statusBusbar);

      final matchComponent =
          selectedComponents.isEmpty ||
          selectedComponents.contains(panel.statusComponent);

      final matchPalet =
          selectedPalet.isEmpty || selectedPalet.contains(panel.statusPalet);

      final matchCorepart =
          selectedCorepart.isEmpty ||
          selectedCorepart.contains(panel.statusCorepart);

      // Jika tidak cocok dengan salah satu filter dasar, langsung sembunyikan.
      final baseFiltersMatch =
          matchSearch &&
          matchPanelVendor &&
          matchBusbarVendor &&
          matchComponentVendor &&
          matchPaletVendor &&
          matchCorepartVendor &&
          matchStatus &&
          matchComponent &&
          matchPalet &&
          matchCorepart;

      if (!baseFiltersMatch) {
        return false;
      }

      final panelStatus = _getPanelFilterStatus(panel);
      final isArchived = panelStatus == PanelFilterStatus.closedArchived;
      if (isArchived) {
        return includeArchived;
      }
      return selectedPanelStatuses.isEmpty ||
          selectedPanelStatuses.contains(panelStatus);
    }).toList();
  }

  List<PanelDisplayData> get filteredPanelsForDisplay {
    var tabFilteredPanels = _panelsAfterPrimaryFilters;
    final role = widget.currentCompany.role; // Dapatkan peran akun saat ini

    switch (_tabController.index) {
      case 0: // All
        // Tidak ada filter tambahan untuk tab "All"
        break;
      case 1: // Open Vendor
        if (role == AppRole.k5) {
          tabFilteredPanels = tabFilteredPanels
              .where((data) => data.busbarVendorIds.isEmpty)
              .toList();
        } else if (role == AppRole.warehouse) {
          tabFilteredPanels = tabFilteredPanels
              .where((data) => data.componentVendorIds.isEmpty)
              .toList();
        } else {
          // admin, k3, or others (default to original logic for consistency)
          tabFilteredPanels = tabFilteredPanels
              .where(
                (data) =>
                    data.busbarVendorIds.isEmpty ||
                    data.componentVendorIds.isEmpty ||
                    data.paletVendorIds.isEmpty ||
                    data.corepartVendorIds.isEmpty,
              )
              .toList();
        }
        break;
      case 2: // On Going Panel
        if (role == AppRole.k5) {
          tabFilteredPanels = tabFilteredPanels
              .where(
                (data) =>
                    data.busbarVendorIds.contains(widget.currentCompany.id) &&
                    (data.panel.percentProgress ?? 0) < 100 &&
                    !data.panel.isClosed,
              )
              .toList();
        } else if (role == AppRole.warehouse) {
          tabFilteredPanels = tabFilteredPanels
              .where(
                (data) =>
                    data.componentVendorIds.contains(
                      widget.currentCompany.id,
                    ) &&
                    (data.panel.percentProgress ?? 0) < 100 &&
                    !data.panel.isClosed,
              )
              .toList();
        } else {
          // admin, k3, or others (default to original logic for consistency)
          tabFilteredPanels = tabFilteredPanels
              .where(
                (data) =>
                    (data.panel.percentProgress ?? 0) < 100 &&
                    !data.panel.isClosed,
              )
              .toList();
        }
        break;
      case 3: // Ready to Delivery
        tabFilteredPanels = tabFilteredPanels
            .where(
              (data) =>
                  (data.panel.percentProgress ?? 0) >= 100 &&
                  !data.panel.isClosed,
            )
            .toList();
        break;
      case 4: // Closed Panel
        tabFilteredPanels = tabFilteredPanels
            .where((data) => data.panel.isClosed)
            .toList();
        break;
    }

    switch (selectedSort) {
      case SortOption.percentageDesc:
        tabFilteredPanels.sort(
          (a, b) => (b.panel.percentProgress ?? 0).compareTo(
            a.panel.percentProgress ?? 0,
          ),
        );
        break;
      case SortOption.percentageAsc:
        tabFilteredPanels.sort(
          (a, b) => (a.panel.percentProgress ?? 0).compareTo(
            b.panel.percentProgress ?? 0,
          ),
        );
        break;
      case SortOption.durationDesc:
        tabFilteredPanels.sort(
          (a, b) => (a.panel.startDate ?? DateTime(1900)).compareTo(
            b.panel.startDate ?? DateTime(1900),
          ),
        );
        break;
      case SortOption.durationAsc:
        tabFilteredPanels.sort(
          (a, b) => (b.panel.startDate ?? DateTime(1900)).compareTo(
            a.panel.startDate ?? DateTime(1900),
          ),
        );
        break;
      case SortOption.panelNoAZ:
        tabFilteredPanels.sort(
          (a, b) => a.panel.noPanel.compareTo(b.panel.noPanel),
        );
        break;
      case SortOption.panelNoZA:
        tabFilteredPanels.sort(
          (a, b) => b.panel.noPanel.compareTo(a.panel.noPanel),
        );
        break;
      default:
        tabFilteredPanels.sort(
          (a, b) => (a.panel.startDate ?? DateTime(1900)).compareTo(
            b.panel.startDate ?? DateTime(1900),
          ),
        );
        break;
    }
    return tabFilteredPanels;
  }

  void _openFilterBottomSheet() {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => PanelFilterBottomSheet(
        selectedStatuses: selectedStatuses,
        selectedComponents: selectedComponents,
        selectedPalet: selectedPalet,
        selectedCorepart: selectedCorepart,
        includeArchived: includeArchived,
        selectedSort: selectedSort,
        selectedPanelStatuses: selectedPanelStatuses,
        allK3Vendors: _allK3Vendors,
        allK5Vendors: _allK5Vendors,
        allWHSVendors: _allWHSVendors,
        selectedPanelVendors: selectedPanelVendors,
        selectedBusbarVendors: selectedBusbarVendors,
        selectedComponentVendors: selectedComponentVendors,
        selectedPaletVendors: selectedPaletVendors,
        selectedCorepartVendors: selectedCorepartVendors,
        onStatusesChanged: (value) => setState(() => selectedStatuses = value),
        onComponentsChanged: (value) =>
            setState(() => selectedComponents = value),
        onPaletChanged: (value) => setState(() => selectedPalet = value),
        onCorepartChanged: (value) => setState(() => selectedCorepart = value),
        onIncludeArchivedChanged: (value) =>
            setState(() => includeArchived = value ?? false),
        onSortChanged: (value) => setState(() => selectedSort = value),
        onPanelStatusesChanged: (value) =>
            setState(() => selectedPanelStatuses = value),
        onPanelVendorsChanged: (value) =>
            setState(() => selectedPanelVendors = value),
        onBusbarVendorsChanged: (value) =>
            setState(() => selectedBusbarVendors = value),
        onComponentVendorsChanged: (value) =>
            setState(() => selectedComponentVendors = value),
        onPaletVendorsChanged: (value) =>
            setState(() => selectedPaletVendors = value),
        onCorepartVendorsChanged: (value) =>
            setState(() => selectedCorepartVendors = value),
        onReset: () {
          setState(() {
            searchQuery = "";
            _searchController.clear();
            includeArchived = false;
            selectedSort = null;
            selectedPanelStatuses = [];
            selectedPanelVendors = [];
            selectedBusbarVendors = [];
            selectedComponentVendors = [];
            selectedPaletVendors = [];
            selectedCorepartVendors = [];
            selectedStatuses = [];
            selectedComponents = [];
            selectedPalet = [];
            selectedCorepart = [];
          });
          Navigator.pop(context);
        },
      ),
    );
  }

  void _openEditPanelBottomSheet(PanelDisplayData dataToEdit) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => EditPanelBottomSheet(
        panelData: dataToEdit,
        currentCompany: widget.currentCompany,
        k3Vendors: _allK3Vendors,
        onSave: (updatedPanel) {
          widget.onRefresh();
        },
        onDelete: () async {
          Navigator.of(context).pop();
          await DatabaseHelper.instance.deletePanel(dataToEdit.panel.noPp);
          widget.onRefresh();
          if (mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(
                content: Text(
                  'Panel "${dataToEdit.panel.noPanel}" berhasil dihapus.',
                ),
                backgroundColor: Colors.green,
                behavior: SnackBarBehavior.floating,
              ),
            );
          }
        },
      ),
    );
  }

  void _openEditStatusBottomSheet(PanelDisplayData dataToEdit) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => EditStatusBottomSheet(
        duration: _formatDuration(dataToEdit.panel.startDate),
        startDate:
            dataToEdit.panel.startDate, // ✅ PASTIKAN ANDA MENAMBAHKAN INI
        progress: (dataToEdit.panel.percentProgress ?? 0) / 100.0,
        panelData: dataToEdit,
        panelVendorName: dataToEdit.panelVendorName,
        busbarVendorName: dataToEdit.busbarVendorNames,
        currentCompany: widget.currentCompany,
        onSave: () {
          widget.onRefresh();
        },
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: _loadInitialData,
      color: AppColors.schneiderGreen,
      child: SafeArea(
        child: _isLoading ? _buildSkeletonView() : _buildContentView(),
      ),
    );
  }

  Widget _buildContentView() {
    final panelsToDisplay = filteredPanelsForDisplay;
    final baseFilteredList = _panelsAfterPrimaryFilters;
    final role = widget.currentCompany.role;

    final allCount = baseFilteredList.length;

    // Logika penghitungan Open Vendor yang spesifik berdasarkan peran
    final int openVendorCount;
    if (role == AppRole.k5) {
      openVendorCount = baseFilteredList
          .where((data) => data.busbarVendorIds.isEmpty)
          .length;
    } else if (role == AppRole.warehouse) {
      openVendorCount = baseFilteredList
          .where((data) => data.componentVendorIds.isEmpty)
          .length;
    } else {
      // admin, k3
      openVendorCount = baseFilteredList
          .where(
            (data) =>
                data.busbarVendorIds.isEmpty ||
                data.componentVendorIds.isEmpty ||
                data.paletVendorIds.isEmpty ||
                data.corepartVendorIds.isEmpty,
          )
          .length;
    }

    // Logika penghitungan On Going Panel yang spesifik berdasarkan peran
    final int onGoingPanelCount;
    if (role == AppRole.k5) {
      onGoingPanelCount = baseFilteredList
          .where(
            (data) =>
                data.busbarVendorIds.contains(widget.currentCompany.id) &&
                (data.panel.percentProgress ?? 0) < 100 &&
                !data.panel.isClosed,
          )
          .length;
    } else if (role == AppRole.warehouse) {
      onGoingPanelCount = baseFilteredList
          .where(
            (data) =>
                data.componentVendorIds.contains(widget.currentCompany.id) &&
                (data.panel.percentProgress ?? 0) < 100 &&
                !data.panel.isClosed,
          )
          .length;
    } else {
      // admin, k3
      onGoingPanelCount = baseFilteredList
          .where(
            (data) =>
                (data.panel.percentProgress ?? 0) < 100 && !data.panel.isClosed,
          )
          .length;
    }

    final readyToDeliveryCount = baseFilteredList
        .where(
          (data) =>
              (data.panel.percentProgress ?? 0) >= 100 && !data.panel.isClosed,
        )
        .length;
    final closedPanelCount = baseFilteredList
        .where((data) => data.panel.isClosed)
        .length;

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                "Alignment Panel, Busbar, dan Komponen",
                style: TextStyle(
                  color: AppColors.black,
                  fontSize: 24,
                  fontWeight: FontWeight.w400,
                ),
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: SearchField(
                      controller: _searchController,
                      onChanged: (value) => setState(() => searchQuery = value),
                    ),
                  ),
                  const SizedBox(width: 12),
                  InkWell(
                    onTap: _openFilterBottomSheet,
                    borderRadius: BorderRadius.circular(12),
                    child: Container(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: AppColors.white,
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(color: AppColors.grayLight),
                      ),
                      child: Image.asset(
                        'assets/images/filter-green.png',
                        width: 20,
                        height: 20,
                        color: AppColors.schneiderGreen,
                      ),
                    ),
                  ),
                ],
              ),
              TabBar(
                controller: _tabController,
                isScrollable: true,
                labelColor: AppColors.black,
                unselectedLabelColor: AppColors.gray,
                indicatorColor: AppColors.schneiderGreen,
                indicatorWeight: 2,
                tabAlignment: TabAlignment.start,
                padding: EdgeInsets.zero,
                indicatorSize: TabBarIndicatorSize.label,
                overlayColor: WidgetStateProperty.all(Colors.transparent),
                dividerColor: Colors.transparent,
                labelStyle: const TextStyle(
                  fontWeight: FontWeight.w500,
                  fontFamily: 'Lexend',
                  fontSize: 12,
                ),
                unselectedLabelStyle: const TextStyle(
                  fontWeight: FontWeight.w400,
                  fontFamily: 'Lexend',
                  fontSize: 12,
                ),
                tabs: [
                  Tab(text: "All ($allCount)"),
                  Tab(text: "Open Vendor ($openVendorCount)"),
                  Tab(text: "Need to Track ($onGoingPanelCount)"),
                  Tab(text: "Ready to Delivery ($readyToDeliveryCount)"),
                  Tab(text: "Closed Panel ($closedPanelCount)"),
                ],
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        Expanded(
          child: panelsToDisplay.isEmpty
              ? const Center(
                  child: Padding(
                    padding: EdgeInsets.symmetric(vertical: 48.0),
                    child: Text(
                      "Tidak ada panel yang ditemukan",
                      style: TextStyle(color: AppColors.gray, fontSize: 14),
                    ),
                  ),
                )
              : ListView.separated(
                  padding: const EdgeInsets.fromLTRB(20, 0, 20, 100),
                  itemCount: panelsToDisplay.length,
                  separatorBuilder: (context, index) =>
                      const SizedBox(height: 16),
                  itemBuilder: (context, index) {
                    final data = panelsToDisplay[index];
                    final panel = data.panel;
                    return PanelProgressCard(
                      duration: _formatDuration(panel.startDate),
                      progress: (panel.percentProgress ?? 0) / 100.0,
                      startDate:
                          panel.startDate, // ✅ PASTIKAN ANDA MENAMBAHKAN INI
                      progressLabel: "${panel.percentProgress?.toInt() ?? 0}%",
                      panelTitle: panel.noPanel,
                      statusBusbar: panel.statusBusbar ?? "N/A",
                      statusComponent: panel.statusComponent ?? "N/A",
                      statusPalet: panel.statusPalet ?? "N/A",
                      statusCorepart: panel.statusCorepart ?? "N/A",
                      ppNumber: panel.noPp,
                      wbsNumber: panel.noWbs,
                      onEdit: () {
                        final role = widget.currentCompany.role;
                        if (role == AppRole.admin || role == AppRole.k3) {
                          _openEditPanelBottomSheet(data);
                        } else if (role == AppRole.k5 ||
                            role == AppRole.warehouse) {
                          _openEditStatusBottomSheet(data);
                        }
                      },
                      panelVendorName: data.panelVendorName,
                      busbarVendorName: data.busbarVendorNames,
                      componentVendorName: data.componentVendorNames,
                      paletVendorName: data.paletVendorNames,
                      corepartVendorName: data.corepartVendorNames,
                      isClosed: panel.isClosed,
                      closedDate: panel.closedDate,
                      busbarRemarks: data.busbarRemarks,
                    );
                  },
                ),
        ),
      ],
    );
  }

  Widget _buildSkeletonView() {
    return SingleChildScrollView(
      physics: const NeverScrollableScrollPhysics(),
      padding: const EdgeInsets.all(20.0),
      child: Shimmer.fromColors(
        baseColor: Colors.grey[200]!,
        highlightColor: Colors.grey[100]!,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildSkeletonBox(height: 28, width: 200),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(child: _buildSkeletonBox(height: 48)),
                const SizedBox(width: 12),
                _buildSkeletonBox(width: 48, height: 48),
              ],
            ),
            const SizedBox(height: 8),
            _buildSkeletonBox(height: 48, width: double.infinity),
            const SizedBox(height: 12),
            ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: 5,
              separatorBuilder: (context, index) => const SizedBox(height: 16),
              itemBuilder: (context, index) => _buildSkeletonCard(),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSkeletonCard() {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.grayLight, width: 1),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              _buildSkeletonBox(width: 100, height: 14),
              _buildSkeletonBox(width: 24, height: 24),
            ],
          ),
          const SizedBox(height: 8),
          _buildSkeletonBox(width: double.infinity, height: 8),
          const SizedBox(height: 12),
          _buildSkeletonBox(width: 150, height: 20),
          const SizedBox(height: 8),
          Row(
            children: [
              _buildSkeletonBox(width: 80, height: 14),
              const SizedBox(width: 10),
              _buildSkeletonBox(width: 80, height: 14),
            ],
          ),
          const SizedBox(height: 10),
          const Divider(),
          const SizedBox(height: 10),
          _buildSkeletonBox(width: 120, height: 12),
        ],
      ),
    );
  }

  Widget _buildSkeletonBox({double? width, double height = 16}) {
    return Container(
      width: width,
      height: height,
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(8),
      ),
    );
  }
}
