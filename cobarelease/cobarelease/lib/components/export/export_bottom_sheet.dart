import 'package:flutter/material.dart';
import 'package:secpanel/helpers/db_helper.dart';
import 'package:secpanel/models/approles.dart';
import 'package:secpanel/models/busbar.dart';
import 'package:secpanel/models/companyaccount.dart';
import 'package:secpanel/models/component.dart';
import 'package:secpanel/models/palet.dart';
import 'package:secpanel/models/corepart.dart';
import 'package:secpanel/models/panels.dart';
import 'package:secpanel/models/company.dart';
import 'package:secpanel/theme/colors.dart';

class PreviewBottomSheet extends StatefulWidget {
  final Company
  currentUser; // Tambahkan ini untuk menerima pengguna yang sedang login

  const PreviewBottomSheet({super.key, required this.currentUser});

  @override
  State<PreviewBottomSheet> createState() => _PreviewBottomSheetState();
}

class _PreviewBottomSheetState extends State<PreviewBottomSheet> {
  bool _isLoading = true;
  late List<Company> _companies;
  late List<CompanyAccount> _companyAccounts;
  late List<Panel> _panels;
  late List<Busbar> _busbars;
  late List<Component> _components;
  late List<Palet> _palet;
  late List<Corepart> _corepart;

  // Initial state for selected tables will be determined by user role
  final Map<String, bool> _selectedTables = {
    'Companies': false, // Default to false, will be set based on role
    'Company Accounts': false,
    'Panels': false,
    'Busbars': false,
    'Components': false,
    'Palet': false,
    'Corepart': false,
  };

  final List<String> _availableTableNames =
      []; // Filtered table names for display
  String _selectedFormat = 'Excel';

  @override
  void initState() {
    super.initState();
    _setAvailableTablesBasedOnRole();
    _loadAllData(); // Load all data, filtering happens in DBHelper
  }

  // Metode untuk mengatur tabel yang tersedia berdasarkan peran pengguna
  void _setAvailableTablesBasedOnRole() {
    _availableTableNames.clear();

    // Semua peran bisa mengekspor Companies dan Company Accounts
    _availableTableNames.add('Companies');
    _availableTableNames.add('Company Accounts');

    // Atur pilihan default berdasarkan peran
    _selectedTables['Companies'] = true;
    _selectedTables['Company Accounts'] = true;

    switch (widget.currentUser.role) {
      case AppRole.admin:
        _availableTableNames.add('Panels');
        _availableTableNames.add('Busbars');
        _availableTableNames.add('Components');
        _availableTableNames.add('Palet');
        _availableTableNames.add('Corepart');
        // Admin memilih semua secara default
        _selectedTables['Panels'] = true;
        _selectedTables['Busbars'] = true;
        _selectedTables['Components'] = true;
        _selectedTables['Palet'] = true;
        _selectedTables['Corepart'] = true;

        break;
      case AppRole.k3:
        _availableTableNames.add('Panels');
        _availableTableNames.add('Busbars');
        _availableTableNames.add(
          'Components',
        ); // K3 perlu melihat komponen untuk panel mereka
        _availableTableNames.add('Palet');
        _availableTableNames.add('Corepart');
        // K3 memilih panels, busbars, components secara default
        _selectedTables['Panels'] = true;
        _selectedTables['Busbars'] = true;
        _selectedTables['Components'] = true;
        _selectedTables['Palet'] = true;
        _selectedTables['Corepart'] = true;
        break;
      case AppRole.k5:
        _availableTableNames.add('Panels');
        _availableTableNames.add('Busbars');
        // K5 memilih panels, busbars secara default
        _selectedTables['Panels'] = true;
        _selectedTables['Busbars'] = true;
        _selectedTables['Components'] = false;

        _selectedTables['Palet'] = false;
        _selectedTables['Corepart'] = false;

        break;
      case AppRole.warehouse:
        _availableTableNames.add('Components');
        // Warehouse memilih komponen secara default
        _selectedTables['Panels'] =
            false; // Warehouse tidak memiliki akses langsung ke data tabel panels
        _selectedTables['Busbars'] =
            false; // Warehouse tidak memiliki akses langsung ke data tabel busbars
        _selectedTables['Components'] = true;

        _selectedTables['Palet'] = false;
        _selectedTables['Corepart'] = false;
        break;
      default:
        // Tidak ada tabel spesifik untuk peran lainnya
        break;
    }
  }

  Future<void> _loadAllData() async {
    final db = DatabaseHelper.instance;
    final filteredData = await db.getFilteredDataForExport(widget.currentUser);

    _companies = filteredData['companies'] as List<Company>;
    _companyAccounts = filteredData['companyAccounts'] as List<CompanyAccount>;
    _panels = filteredData['panels'] as List<Panel>;
    _busbars = filteredData['busbars'] as List<Busbar>;
    _components = filteredData['components'] as List<Component>;
    _palet = filteredData['palet'] as List<Palet>;
    _corepart = filteredData['corepart'] as List<Corepart>;

    if (mounted) {
      setState(() => _isLoading = false);
    }
  }

  // Metode untuk menampilkan pratinjau data
  void _showPreview() {
    if (_isLoading) return;

    // Filter tabel yang akan ditampilkan di pratinjau berdasarkan opsi yang dipilih di bottom sheet
    final List<String> tablesToShowInPreview = _availableTableNames
        .where((name) => _selectedTables[name] == true)
        .toList();

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (context) => _DataPreviewerSheet(
        companies: _companies,
        companyAccounts: _companyAccounts,
        panels: _panels,
        busbars: _busbars,
        components: _components,
        palet: _palet,
        corepart: _corepart,
        tableNames:
            tablesToShowInPreview, // Gunakan daftar yang difilter untuk pratinjau
      ),
    );
  }

  // Metode pembangun untuk judul bagian
  Widget _buildSectionTitle(String title) {
    return Padding(
      padding: const EdgeInsets.only(top: 24.0, bottom: 12.0),
      child: Text(
        title,
        style: const TextStyle(
          fontFamily: 'Lexend',
          fontSize: 14,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }

  // Metode pembangun untuk opsi multi-pilihan (checkbox)
  Widget _buildMultiSelectOption(String label) {
    final bool isSelected = _selectedTables[label] ?? false;
    return GestureDetector(
      onTap: () {
        setState(() {
          _selectedTables[label] = !isSelected;
        });
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        margin: const EdgeInsets.only(right: 8, bottom: 12),
        decoration: BoxDecoration(
          color: isSelected
              ? AppColors.schneiderGreen.withOpacity(0.08)
              : Colors.white,
          border: Border.all(
            color: isSelected ? AppColors.schneiderGreen : AppColors.grayLight,
          ),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          label,
          style: const TextStyle(fontWeight: FontWeight.w400, fontSize: 12),
        ),
      ),
    );
  }

  // Metode pembangun untuk opsi satu pilihan (radio button)
  Widget _buildSingleSelectOption(String format) {
    final bool isSelected = _selectedFormat == format;
    return GestureDetector(
      onTap: () {
        setState(() {
          _selectedFormat = format;
        });
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        margin: const EdgeInsets.only(right: 8, bottom: 12),
        decoration: BoxDecoration(
          color: isSelected
              ? AppColors.schneiderGreen.withOpacity(0.08)
              : Colors.white,
          border: Border.all(
            color: isSelected ? AppColors.schneiderGreen : AppColors.grayLight,
          ),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          format,
          style: const TextStyle(fontWeight: FontWeight.w400, fontSize: 12),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final bool isAnyTableSelected = _selectedTables.values.any(
      (isSelected) => isSelected,
    );

    return Padding(
      padding: EdgeInsets.only(
        left: 20,
        right: 20,
        top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 20,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              height: 5,
              width: 40,
              decoration: BoxDecoration(
                color: AppColors.grayLight,
                borderRadius: BorderRadius.circular(100),
              ),
            ),
          ),
          const SizedBox(height: 24),
          const Text(
            "Ekspor Data",
            style: TextStyle(fontSize: 24, fontWeight: FontWeight.w400),
          ),
          Flexible(
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _buildSectionTitle("Pilih Tabel untuk Diekspor"),
                  Wrap(
                    children:
                        _availableTableNames // Gunakan daftar yang sudah difilter di sini
                            .map((name) => _buildMultiSelectOption(name))
                            .toList(),
                  ),
                  _buildSectionTitle("Pilih Format File"),
                  Wrap(
                    children: ['Excel', 'JSON']
                        .map((format) => _buildSingleSelectOption(format))
                        .toList(),
                  ),
                  const SizedBox(height: 24),
                  SizedBox(
                    width: double.infinity,
                    child: OutlinedButton.icon(
                      icon: const Icon(Icons.visibility_outlined, size: 18),
                      label: const Text(
                        "Lihat Pratinjau Data",
                        style: TextStyle(
                          fontWeight: FontWeight.w500,
                          fontSize: 12,
                        ),
                      ),
                      onPressed: _showPreview,
                      style: OutlinedButton.styleFrom(
                        foregroundColor: AppColors.schneiderGreen,
                        side: const BorderSide(color: AppColors.grayLight),
                        padding: const EdgeInsets.symmetric(vertical: 14),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 24),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    side: const BorderSide(color: AppColors.schneiderGreen),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text(
                    'Batal',
                    style: TextStyle(
                      color: AppColors.schneiderGreen,
                      fontSize: 12,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: ElevatedButton(
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 16),
                    backgroundColor: AppColors.schneiderGreen,
                    foregroundColor: Colors.white,
                    disabledBackgroundColor: AppColors.grayNeutral,
                    elevation: 0,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                  onPressed: !isAnyTableSelected
                      ? null
                      : () {
                          Navigator.of(context).pop({
                            'tables': _selectedTables,
                            'format': _selectedFormat,
                          });
                        },
                  child: const Text('Ekspor', style: TextStyle(fontSize: 12)),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _DataPreviewerSheet extends StatefulWidget {
  final List<Company> companies;
  final List<CompanyAccount> companyAccounts;
  final List<Panel> panels;
  final List<Busbar> busbars;
  final List<Component> components;
  final List<Palet> palet;
  final List<Corepart> corepart;

  final List<String>
  tableNames; // Daftar tabel yang benar-benar akan ditampilkan

  const _DataPreviewerSheet({
    required this.companies,
    required this.companyAccounts,
    required this.panels,
    required this.busbars,
    required this.components,
    required this.palet,
    required this.corepart,
    required this.tableNames,
  });

  @override
  State<_DataPreviewerSheet> createState() => _DataPreviewerSheetState();
}

class _DataPreviewerSheetState extends State<_DataPreviewerSheet>
    with TickerProviderStateMixin {
  late final TabController _tabController;

  final TextStyle headerStyle = const TextStyle(
    fontFamily: 'Lexend',
    fontWeight: FontWeight.w600,
    color: AppColors.black,
    fontSize: 12,
  );

  final TextStyle cellStyle = const TextStyle(
    fontFamily: 'Lexend',
    fontWeight: FontWeight.w400,
    color: AppColors.gray,
    fontSize: 12,
  );

  @override
  void initState() {
    super.initState();
    _tabController = TabController(
      length: widget
          .tableNames
          .length, // Sesuaikan panjang tab dengan tabel yang akan ditampilkan
      vsync: this,
    );
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    // Mapping antara nama tabel string dan widget pembangun tabel yang sesuai
    Map<String, Widget Function()> tableBuilders = {
      'Companies': _buildCompaniesTable,
      'Company Accounts': _buildCompanyAccountsTable,
      'Panels': _buildPanelsTable,
      'Busbars': _buildBusbarsTable,
      'Components': _buildComponentsTable,
      'Palet': _buildPaletTable,
      'Corepart': _buildCorepartTable,
    };

    // Filter daftar widget TabBarView berdasarkan tableNames yang diberikan
    List<Widget> tabViews = widget.tableNames
        .map((tableName) => tableBuilders[tableName]!())
        .toList();

    return ConstrainedBox(
      constraints: BoxConstraints(
        maxHeight: MediaQuery.of(context).size.height * 0.85,
      ),
      child: Padding(
        padding: const EdgeInsets.only(left: 20, right: 20, top: 16),
        child: Column(
          children: [
            Center(
              child: Container(
                height: 5,
                width: 40,
                decoration: BoxDecoration(
                  color: AppColors.grayLight,
                  borderRadius: BorderRadius.circular(100),
                ),
              ),
            ),
            const SizedBox(height: 24),
            const Text(
              "Pratinjau Data",
              style: TextStyle(fontSize: 24, fontWeight: FontWeight.w400),
            ),
            const SizedBox(height: 16),
            TabBar(
              controller: _tabController,
              isScrollable: true,
              labelColor: AppColors.black,
              unselectedLabelColor: AppColors.gray,
              indicatorColor: AppColors.schneiderGreen,
              indicatorWeight: 2.5,
              tabAlignment: TabAlignment.start,
              padding: EdgeInsets.zero,
              tabs: widget.tableNames.map((name) => Tab(text: name)).toList(),
            ),
            const Divider(height: 1, color: AppColors.grayLight),
            Expanded(
              child: TabBarView(
                controller: _tabController,
                children:
                    tabViews, // Gunakan daftar tabViews yang sudah difilter
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTableContainer(Widget dataTable) {
    return SingleChildScrollView(
      padding: const EdgeInsets.only(top: 8.0, bottom: 20.0),
      scrollDirection: Axis.vertical,
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: dataTable,
      ),
    );
  }

  DataTable _buildStyledDataTable({
    required List<DataColumn> columns,
    required List<DataRow> rows,
  }) {
    return DataTable(
      headingRowColor: WidgetStateProperty.all(
        AppColors.schneiderGreen.withOpacity(0.05),
      ),
      headingRowHeight: 40,
      dataRowMinHeight: 42,
      dataRowMaxHeight: 48,
      columnSpacing: 32,
      border: const TableBorder(
        horizontalInside: BorderSide(color: AppColors.grayLight, width: 1),
        bottom: BorderSide(color: AppColors.grayLight, width: 1),
      ),
      columns: columns,
      rows: rows,
    );
  }

  Widget _buildCompaniesTable() {
    return _buildTableContainer(
      _buildStyledDataTable(
        columns: [
          DataColumn(label: Text('ID', style: headerStyle)),
          DataColumn(label: Text('Name', style: headerStyle)),
          DataColumn(label: Text('Role', style: headerStyle)),
        ],
        rows: widget.companies
            .map(
              (company) => DataRow(
                cells: [
                  DataCell(Text(company.id, style: cellStyle)),
                  DataCell(Text(company.name, style: cellStyle)),
                  DataCell(Text(company.role.name, style: cellStyle)),
                ],
              ),
            )
            .toList(),
      ),
    );
  }

  Widget _buildCompanyAccountsTable() {
    return _buildTableContainer(
      _buildStyledDataTable(
        columns: [
          DataColumn(label: Text('Username', style: headerStyle)),
          DataColumn(label: Text('Password', style: headerStyle)),
          DataColumn(label: Text('Company ID', style: headerStyle)),
        ],
        rows: widget.companyAccounts
            .map(
              (account) => DataRow(
                cells: [
                  DataCell(Text(account.username, style: cellStyle)),
                  DataCell(Text('••••••••', style: cellStyle)),
                  DataCell(Text(account.companyId, style: cellStyle)),
                ],
              ),
            )
            .toList(),
      ),
    );
  }

  Widget _buildPanelsTable() {
    return _buildTableContainer(
      _buildStyledDataTable(
        columns: [
          DataColumn(label: Text('No PP', style: headerStyle)),
          DataColumn(label: Text('No Panel', style: headerStyle)),
          DataColumn(label: Text('Vendor ID', style: headerStyle)),
        ],
        rows: widget.panels
            .map(
              (panel) => DataRow(
                cells: [
                  DataCell(Text(panel.noPp, style: cellStyle)),
                  DataCell(Text(panel.noPanel, style: cellStyle)),
                  DataCell(Text(panel.vendorId ?? 'N/A', style: cellStyle)),
                ],
              ),
            )
            .toList(),
      ),
    );
  }

  Widget _buildBusbarsTable() {
    return _buildTableContainer(
      _buildStyledDataTable(
        columns: [
          DataColumn(label: Text('Panel No PP', style: headerStyle)),
          DataColumn(label: Text('Vendor', style: headerStyle)),
        ],
        rows: widget.busbars
            .map(
              (busbar) => DataRow(
                cells: [
                  DataCell(Text(busbar.panelNoPp, style: cellStyle)),
                  DataCell(Text(busbar.vendor, style: cellStyle)),
                ],
              ),
            )
            .toList(),
      ),
    );
  }

  Widget _buildComponentsTable() {
    return _buildTableContainer(
      _buildStyledDataTable(
        columns: [
          DataColumn(label: Text('Panel No PP', style: headerStyle)),
          DataColumn(label: Text('Vendor', style: headerStyle)),
        ],
        rows: widget.components
            .map(
              (component) => DataRow(
                cells: [
                  DataCell(Text(component.panelNoPp, style: cellStyle)),
                  DataCell(Text(component.vendor, style: cellStyle)),
                ],
              ),
            )
            .toList(),
      ),
    );
  }

  Widget _buildPaletTable() {
    return _buildTableContainer(
      _buildStyledDataTable(
        columns: [
          DataColumn(label: Text('Panel No PP', style: headerStyle)),
          DataColumn(label: Text('Vendor', style: headerStyle)),
        ],
        rows: widget.palet
            .map(
              (palet) => DataRow(
                cells: [
                  DataCell(Text(palet.panelNoPp, style: cellStyle)),
                  DataCell(Text(palet.vendor, style: cellStyle)),
                ],
              ),
            )
            .toList(),
      ),
    );
  }

  Widget _buildCorepartTable() {
    return _buildTableContainer(
      _buildStyledDataTable(
        columns: [
          DataColumn(label: Text('Panel No PP', style: headerStyle)),
          DataColumn(label: Text('Vendor', style: headerStyle)),
        ],
        rows: widget.corepart
            .map(
              (corepart) => DataRow(
                cells: [
                  DataCell(Text(corepart.panelNoPp, style: cellStyle)),
                  DataCell(Text(corepart.vendor, style: cellStyle)),
                ],
              ),
            )
            .toList(),
      ),
    );
  }
}
