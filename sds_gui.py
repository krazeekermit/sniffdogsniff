from PyQt5.QtWidgets import *
from PyQt5 import QtGui
from PyQt5.QtCore import *
import sys
import sds


class PandasTable(QAbstractTableModel):
    def __init__(self, data):
        QAbstractTableModel.__init__(self)
        self._data_ = data

    def rowCount(self, parent=None):
        return self._data_.shape[0]

    def columnCount(self, parent=None):
        return self._data_.shape[1]

    def data(self, index, role=Qt.DisplayRole):
        if index.isValid():
            if role == Qt.DisplayRole:
                return str(self._data_.iloc[index.row(), index.column()])
        return None

    def headerData(self, col, orientation, role):
        if orientation == Qt.Horizontal and role == Qt.DisplayRole:
            return self._data_.columns[col]
        return None


class SniffDogSniffUi(QMainWindow):
    def __init__(self, data_widget):
        super().__init__()
        self.setMinimumHeight(500)
        self.setMinimumWidth(900)
        self.setWindowTitle("SniffDogSniff")
        self.setWindowIcon(QtGui.QIcon('./sniffdogsniff_icon.png'))
        self._initialize_menu_bar_()
        self._central_widget_ = SDSCentralWidget(data_widget)
        self.setCentralWidget(self._central_widget_)

    def _initialize_menu_bar_(self):
        m_bar = self.menuBar()
        file_menu = m_bar.addMenu('&File')
        file_menu_exit_action = QAction('&Exit', self)
        file_menu_save_action = QAction('&Save', self)
        file_menu.addAction(file_menu_save_action)
        file_menu.addAction(file_menu_exit_action)

        help_menu = m_bar.addMenu('&Help')
        help_menu_about_action = QAction('&About', self)
        help_menu.addAction(help_menu_about_action)


class SDSCentralWidget(QWidget):
    def __init__(self, data_widget):
        super().__init__()
        self._main_layout_ = QVBoxLayout()
        self.setLayout(self._main_layout_)
        query_bar = SDSQueryBar()
        self._main_layout_.addWidget(query_bar)
        self._main_layout_.addWidget(data_widget)


class SDSDataWidget(QWidget):
    def __init__(self):
        super().__init__()
        self._main_layout_ = QHBoxLayout()
        self.setLayout(self._main_layout_)
        self._queries_list_view_ = QListView()
        self._queries_list_view_.setFixedWidth(200)
        self._main_layout_.addWidget(self._queries_list_view_)
        self._data_table_ = QTableView()
        self._main_layout_.addWidget(self._data_table_)
        self._queries_list_view_.clicked.connect(self._searches_list_item_selected)

    def _searches_list_item_selected(self, index: QModelIndex):
        change_table_data(search_data_models[index.row()])

    @property
    def get_table(self):
        return self._data_table_

    @property
    def get_searches_list(self):
        return self._queries_list_view_

    def update_table(self, df):
        self._data_table_.setVisible(False)
        self._data_table_.get_table.setModel(df)
        self._data_table_.update()
        self._data_table_.setVisible(True)


class SDSQueryBar(QWidget):
    def __init__(self):
        super().__init__()
        self._search_button_ = QPushButton('Search')
        self._search_button_.pressed.connect(self._search_button_pressed_)
        self._search_button_.setEnabled(False)
        self._query_field_ = QLineEdit()
        self._query_field_.textChanged.connect(self._query_field_text_changes_)
        self._query_field_.returnPressed.connect(self._query_field_enter_pressed_)
        self._query_field_.setPlaceholderText('Type your queries comma separated')
        self._merge_checkbox_ = QCheckBox('Merge results')
        self._main_layout_ = QHBoxLayout()

        self._main_layout_.addWidget(self._query_field_)
        self._main_layout_.addWidget(self._search_button_)
        self._main_layout_.addWidget(self._merge_checkbox_)
        self.setLayout(self._main_layout_)

    def _query_field_text_changes_(self, text):
        if text != '':
            self._search_button_.setEnabled(True)
        else:
            self._search_button_.setEnabled(False)

    def _merge_checks(self):
        print('checked me!')

    def _query_field_enter_pressed_(self, text):
        perform_searches_fill_data(text)
        print('perform searches')

    def _search_button_pressed_(self):
        self._query_field_enter_pressed_(self._query_field_.text())


def perform_searches_fill_data(queries: str):
    searches = sds.perform_searches(queries, config, 100, 'NORMAL')
    query_list = queries.split(',')
    list_model = QStringListModel(query_list)
    for q in query_list:
        search_data_models.append(PandasTable(searches[query_list.index(q)]))

    data_container_widget.get_searches_list.setModel(list_model)
    change_table_data(PandasTable(searches[0]))


def change_table_data(data_model: PandasTable):
    data_container_widget.get_table.setModel(data_model)


if __name__ == '__main__':
    app = QApplication(sys.argv)
    data_container_widget = SDSDataWidget()
    window = SniffDogSniffUi(data_container_widget)
    config = sds.get_config('./engines.json')
    search_data_models = list()

    window.show()
    app.exec()
